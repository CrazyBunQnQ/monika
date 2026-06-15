package dap

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// DapSession
// ---------------------------------------------------------------------------
// DapSession manages a single debug session's lifecycle and state.
type DapSession struct {
	id                     string
	adapter                *DapResolvedAdapter
	cwd                    string
	program                string
	client                 *DapClient
	status                 DapSessionStatus
	launchedAt             time.Time
	lastUsedAt             time.Time
	mu                     sync.Mutex
	breakpoints            map[string][]DapBreakpointRecord
	functionBreakpoints    []DapFunctionBreakpointRecord
	instructionBreakpoints []DapInstructionBreakpointRecord
	dataBreakpoints        []DapDataBreakpointRecord
	stop                   DapStopLocation
	threads                []DapThread
	lastStackFrames        []DapStackFrame
	output                 strings.Builder
	outputTruncated        bool
	capabilities           *DapCapabilities
	initializedSeen        bool
	needsConfigurationDone bool
	configurationDoneSent  bool
	exitCode               *int
	launchErrCh            chan error
	stoppedCh              chan struct{}
}

// newDapSession creates a new DapSession and registers event handlers on the
// client for lifecycle events (stopped, continued, output, exited,
// terminated, initialized).
func newDapSession(id string, client *DapClient, adapter *DapResolvedAdapter, cwd string, program string) *DapSession {
	s := &DapSession{
		id:          id,
		adapter:     adapter,
		cwd:         cwd,
		program:     program,
		client:      client,
		status:      DapStatusLaunching,
		launchedAt:  time.Now(),
		lastUsedAt:  time.Now(),
		breakpoints: make(map[string][]DapBreakpointRecord),
		stoppedCh:   make(chan struct{}, 1),
	}
	if client != nil {
		// --- stopped ---
		client.OnEvent(DapEventStopped, func(body interface{}, _ *DapEventMessage) {
			var ev DapStoppedEventBody
			if raw, ok := body.(json.RawMessage); ok && len(raw) > 0 {
				_ = json.Unmarshal(raw, &ev)
			}
			s.mu.Lock()
			s.status = DapStatusStopped
			s.stop = DapStopLocation{
				ThreadID: ev.ThreadId,
				Reason:   ev.Reason,
				Text:     ev.Text,
			}
			s.mu.Unlock()
			select {
			case s.stoppedCh <- struct{}{}:
			default:
			}
		})
		// --- continued ---
		client.OnEvent(DapEventContinued, func(body interface{}, _ *DapEventMessage) {
			s.mu.Lock()
			s.status = DapStatusRunning
			s.mu.Unlock()
		})
		// --- output ---
		client.OnEvent(DapEventOutput, func(body interface{}, _ *DapEventMessage) {
			var ev DapOutputEventBody
			if raw, ok := body.(json.RawMessage); ok && len(raw) > 0 {
				_ = json.Unmarshal(raw, &ev)
			}
			s.mu.Lock()
			if !s.outputTruncated {
				n := s.output.Len() + len(ev.Output)
				if n > MaxOutputBytes {
					s.output.Reset()
					s.outputTruncated = true
				} else {
					s.output.WriteString(ev.Output)
				}
			}
			s.mu.Unlock()
		})
		// --- exited ---
		client.OnEvent(DapEventExited, func(body interface{}, _ *DapEventMessage) {
			var ev DapExitedEventBody
			if raw, ok := body.(json.RawMessage); ok && len(raw) > 0 {
				_ = json.Unmarshal(raw, &ev)
			}
			s.mu.Lock()
			s.exitCode = &ev.ExitCode
			s.mu.Unlock()
		})
		// --- terminated ---
		client.OnEvent(DapEventTerminated, func(_ interface{}, _ *DapEventMessage) {
			s.mu.Lock()
			s.status = DapStatusTerminated
			s.mu.Unlock()
			select {
			case s.stoppedCh <- struct{}{}:
			default:
			}
		})
		// --- initialized ---
		client.OnEvent(DapEventInitialized, func(_ interface{}, _ *DapEventMessage) {
			s.mu.Lock()
			s.initializedSeen = true
			caps := client.Capabilities()
			if caps != nil {
				s.needsConfigurationDone = caps.SupportsConfigurationDoneRequest
			}
			s.mu.Unlock()
		})
	}
	return s
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------
// Summary returns a read-only snapshot of the session state.
func (s *DapSession) Summary() DapSessionSummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	adapterName := ""
	if s.adapter != nil {
		adapterName = s.adapter.Name
	}
	hasExited := s.exitCode != nil
	threads := make([]DapThread, len(s.threads))
	copy(threads, s.threads)
	var stopLoc *DapStopLocation
	if s.stop.ThreadID > 0 || s.stop.Reason != "" {
		loc := s.stop
		stopLoc = &loc
	}
	return DapSessionSummary{
		ID:           s.id,
		Status:       s.status,
		IsLocal:      true,
		Adapter:      adapterName,
		ProcessName:  s.program,
		StopLocation: stopLoc,
		Threads:      threads,
		HasExited:    hasExited,
	}
}

// Touch updates the last-used timestamp.
func (s *DapSession) Touch() {
	s.mu.Lock()
	s.lastUsedAt = time.Now()
	s.mu.Unlock()
}

// Capabilities returns the adapter capabilities, if known.
func (s *DapSession) Capabilities() *DapCapabilities {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.capabilities
}

// ---------------------------------------------------------------------------
// Breakpoints
// ---------------------------------------------------------------------------
// SetBreakpoint sets a source breakpoint at the given file and line.  It
// deduplicates by line, sorts the list, and sends the full breakpoint list
// for that file to the adapter.
func (s *DapSession) SetBreakpoint(file string, line int, condition string, timeout time.Duration) ([]DapBreakpointRecord, error) {
	s.mu.Lock()
	existing := s.breakpoints[file]
	// Deduplicate: replace any existing entry at the same line.
	found := false
	for i, bp := range existing {
		if bp.Breakpoint.Line == line {
			existing[i].Breakpoint.Condition = condition
			found = true
			break
		}
	}
	if !found {
		existing = append(existing, DapBreakpointRecord{
			Breakpoint: DapSourceBreakpoint{
				Line:      line,
				Condition: condition,
			},
		})
	}
	sort.Slice(existing, func(i, j int) bool {
		return existing[i].Breakpoint.Line < existing[j].Breakpoint.Line
	})
	s.breakpoints[file] = existing
	// Build the source breakpoint list.
	bps := make([]DapSourceBreakpoint, len(existing))
	for i, bp := range existing {
		bps[i] = bp.Breakpoint
	}
	s.mu.Unlock()
	// Send to adapter.
	args := DapSetBreakpointsArguments{
		Source:      &DapSource{Path: file},
		Breakpoints: bps,
	}
	body, err := s.client.SendRequest("setBreakpoints", args, timeout)
	if err != nil {
		return nil, fmt.Errorf("dap: setBreakpoints: %w", err)
	}
	raw, ok := body.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("dap: setBreakpoints response has no body")
	}
	var resp DapSetBreakpointsResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("dap: unmarshal setBreakpoints response: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Update local records with adapter response.
	records := s.breakpoints[file]
	for i := range resp.Breakpoints {
		if i < len(records) {
			records[i].ID = resp.Breakpoints[i].Id
			records[i].Verified = resp.Breakpoints[i].Verified
			if resp.Breakpoints[i].Line > 0 {
				records[i].ActualLine = resp.Breakpoints[i].Line
			}
		}
	}
	s.breakpoints[file] = records
	result := make([]DapBreakpointRecord, len(records))
	copy(result, records)
	return result, nil
}

// RemoveBreakpoint removes a source breakpoint at the given file and line
// and updates the adapter.
func (s *DapSession) RemoveBreakpoint(file string, line int, timeout time.Duration) ([]DapBreakpointRecord, error) {
	s.mu.Lock()
	existing := s.breakpoints[file]
	updated := make([]DapBreakpointRecord, 0, len(existing))
	for _, bp := range existing {
		if bp.Breakpoint.Line != line {
			updated = append(updated, bp)
		}
	}
	s.breakpoints[file] = updated
	bps := make([]DapSourceBreakpoint, len(updated))
	for i, bp := range updated {
		bps[i] = bp.Breakpoint
	}
	s.mu.Unlock()
	args := DapSetBreakpointsArguments{
		Source:      &DapSource{Path: file},
		Breakpoints: bps,
	}
	body, err := s.client.SendRequest("setBreakpoints", args, timeout)
	if err != nil {
		return nil, fmt.Errorf("dap: setBreakpoints (remove): %w", err)
	}
	raw, ok := body.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("dap: setBreakpoints response has no body")
	}
	var resp DapSetBreakpointsResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("dap: unmarshal setBreakpoints response: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	records := s.breakpoints[file]
	for i := range resp.Breakpoints {
		if i < len(records) {
			records[i].ID = resp.Breakpoints[i].Id
			records[i].Verified = resp.Breakpoints[i].Verified
			if resp.Breakpoints[i].Line > 0 {
				records[i].ActualLine = resp.Breakpoints[i].Line
			}
		}
	}
	s.breakpoints[file] = records
	result := make([]DapBreakpointRecord, len(records))
	copy(result, records)
	return result, nil
}

// SetFunctionBreakpoint sets a breakpoint on a function by name.  It
// deduplicates by name and sends the full list to the adapter.
func (s *DapSession) SetFunctionBreakpoint(name string, condition string, timeout time.Duration) ([]DapFunctionBreakpointRecord, error) {
	s.mu.Lock()
	existing := s.functionBreakpoints
	found := false
	for i, fb := range existing {
		if fb.Breakpoint.Name == name {
			existing[i].Breakpoint.Condition = condition
			found = true
			break
		}
	}
	if !found {
		existing = append(existing, DapFunctionBreakpointRecord{
			Breakpoint: DapFunctionBreakpoint{
				Name:      name,
				Condition: condition,
			},
		})
	}
	s.functionBreakpoints = existing
	bps := make([]DapFunctionBreakpoint, len(existing))
	for i, fb := range existing {
		bps[i] = fb.Breakpoint
	}
	s.mu.Unlock()
	args := DapSetFunctionBreakpointsArguments{Breakpoints: bps}
	body, err := s.client.SendRequest("setFunctionBreakpoints", args, timeout)
	if err != nil {
		return nil, fmt.Errorf("dap: setFunctionBreakpoints: %w", err)
	}
	raw, ok := body.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("dap: setFunctionBreakpoints response has no body")
	}
	var resp struct {
		Breakpoints []DapBreakpoint `json:"breakpoints"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("dap: unmarshal setFunctionBreakpoints response: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	records := s.functionBreakpoints
	for i := range resp.Breakpoints {
		if i < len(records) {
			records[i].ID = resp.Breakpoints[i].Id
			records[i].Verified = resp.Breakpoints[i].Verified
		}
	}
	s.functionBreakpoints = records
	result := make([]DapFunctionBreakpointRecord, len(records))
	copy(result, records)
	return result, nil
}

// RemoveFunctionBreakpoint removes a function breakpoint by name and updates
// the adapter.
func (s *DapSession) RemoveFunctionBreakpoint(name string, timeout time.Duration) ([]DapFunctionBreakpointRecord, error) {
	s.mu.Lock()
	existing := s.functionBreakpoints
	updated := make([]DapFunctionBreakpointRecord, 0, len(existing))
	for _, fb := range existing {
		if fb.Breakpoint.Name != name {
			updated = append(updated, fb)
		}
	}
	s.functionBreakpoints = updated
	bps := make([]DapFunctionBreakpoint, len(updated))
	for i, fb := range updated {
		bps[i] = fb.Breakpoint
	}
	s.mu.Unlock()
	args := DapSetFunctionBreakpointsArguments{Breakpoints: bps}
	body, err := s.client.SendRequest("setFunctionBreakpoints", args, timeout)
	if err != nil {
		return nil, fmt.Errorf("dap: setFunctionBreakpoints (remove): %w", err)
	}
	raw, ok := body.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("dap: setFunctionBreakpoints response has no body")
	}
	var resp struct {
		Breakpoints []DapBreakpoint `json:"breakpoints"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("dap: unmarshal setFunctionBreakpoints response: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	records := s.functionBreakpoints
	for i := range resp.Breakpoints {
		if i < len(records) {
			records[i].ID = resp.Breakpoints[i].Id
			records[i].Verified = resp.Breakpoints[i].Verified
		}
	}
	s.functionBreakpoints = records
	result := make([]DapFunctionBreakpointRecord, len(records))
	copy(result, records)
	return result, nil
}

// ---------------------------------------------------------------------------
// Execution control
// ---------------------------------------------------------------------------
// Continue resumes execution after a stop.
func (s *DapSession) Continue(timeout time.Duration) (*DapContinueOutcome, error) {
	return s.step("continue", timeout)
}

// StepOver steps over the current source line.
func (s *DapSession) StepOver(timeout time.Duration) (*DapContinueOutcome, error) {
	return s.step("next", timeout)
}

// StepIn steps into the current source line.
func (s *DapSession) StepIn(timeout time.Duration) (*DapContinueOutcome, error) {
	return s.step("stepIn", timeout)
}

// StepOut steps out of the current function.
func (s *DapSession) StepOut(timeout time.Duration) (*DapContinueOutcome, error) {
	return s.step("stepOut", timeout)
}

// Pause pauses the debuggee and fetches the top stack frame.
func (s *DapSession) Pause(timeout time.Duration) (*DapSessionSummary, error) {
	threadID, err := s.resolveThreadID(timeout)
	if err != nil {
		return nil, fmt.Errorf("dap: pause: %w", err)
	}
	args := DapPauseArguments{ThreadId: threadID}
	if _, err := s.client.SendRequest("pause", args, timeout); err != nil {
		return nil, fmt.Errorf("dap: pause: %w", err)
	}
	s.mu.Lock()
	s.status = DapStatusStopped
	s.stop.ThreadID = threadID
	s.mu.Unlock()
	s.fetchTopFrame(timeout)
	summary := s.Summary()
	return &summary, nil
}

// ---------------------------------------------------------------------------
// Introspection
// ---------------------------------------------------------------------------
// StackTrace fetches the stack trace for the stopped thread.
func (s *DapSession) StackTrace(levels int, timeout time.Duration) ([]DapStackFrame, error) {
	threadID, err := s.resolveThreadID(timeout)
	if err != nil {
		return nil, fmt.Errorf("dap: stackTrace: %w", err)
	}
	args := DapStackTraceArguments{
		ThreadId: threadID,
		Levels:   levels,
	}
	body, err := s.client.SendRequest("stackTrace", args, timeout)
	if err != nil {
		return nil, fmt.Errorf("dap: stackTrace: %w", err)
	}
	raw, ok := body.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("dap: stackTrace response has no body")
	}
	var resp DapStackTraceResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("dap: unmarshal stackTrace response: %w", err)
	}
	s.mu.Lock()
	s.lastStackFrames = resp.StackFrames
	s.mu.Unlock()
	return resp.StackFrames, nil
}

// Scopes fetches the variable scopes for the given stack frame.
func (s *DapSession) Scopes(frameID int, timeout time.Duration) ([]DapScope, error) {
	args := DapScopesArguments{FrameId: frameID}
	body, err := s.client.SendRequest("scopes", args, timeout)
	if err != nil {
		return nil, fmt.Errorf("dap: scopes: %w", err)
	}
	raw, ok := body.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("dap: scopes response has no body")
	}
	var resp DapScopesResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("dap: unmarshal scopes response: %w", err)
	}
	return resp.Scopes, nil
}

// Variables fetches the children of the given variable reference.
func (s *DapSession) Variables(variableRef int, timeout time.Duration) ([]DapVariable, error) {
	args := DapVariablesArguments{VariablesReference: variableRef}
	body, err := s.client.SendRequest("variables", args, timeout)
	if err != nil {
		return nil, fmt.Errorf("dap: variables: %w", err)
	}
	raw, ok := body.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("dap: variables response has no body")
	}
	var resp DapVariablesResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("dap: unmarshal variables response: %w", err)
	}
	return resp.Variables, nil
}

// Evaluate evaluates an expression in the context of the given frame.
func (s *DapSession) Evaluate(expression string, context string, frameID int, timeout time.Duration) (*DapEvaluateResponse, error) {
	args := DapEvaluateArguments{
		Expression: expression,
		FrameId:    frameID,
		Context:    context,
	}
	body, err := s.client.SendRequest("evaluate", args, timeout)
	if err != nil {
		return nil, fmt.Errorf("dap: evaluate: %w", err)
	}
	raw, ok := body.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("dap: evaluate response has no body")
	}
	var resp DapEvaluateResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("dap: unmarshal evaluate response: %w", err)
	}
	return &resp, nil
}

// Threads fetches the current thread list from the adapter.
func (s *DapSession) Threads(timeout time.Duration) ([]DapThread, error) {
	body, err := s.client.SendRequest("threads", nil, timeout)
	if err != nil {
		return nil, fmt.Errorf("dap: threads: %w", err)
	}
	raw, ok := body.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("dap: threads response has no body")
	}
	var resp DapThreadsResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("dap: unmarshal threads response: %w", err)
	}
	s.mu.Lock()
	s.threads = resp.Threads
	s.mu.Unlock()
	return resp.Threads, nil
}

// GetOutput returns the accumulated output from the debuggee.
func (s *DapSession) GetOutput() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.output.String()
}

// ---------------------------------------------------------------------------
// Termination
// ---------------------------------------------------------------------------
// Terminate sends "terminate" and "disconnect" requests, then disposes the
// underlying client.
func (s *DapSession) Terminate(timeout time.Duration) {
	// Attempt terminate.
	_, _ = s.client.SendRequest("terminate", nil, timeout)
	// Send disconnect.
	_, _ = s.client.SendRequest("disconnect", nil, timeout)
	s.mu.Lock()
	s.status = DapStatusTerminated
	s.mu.Unlock()
	s.client.Dispose()
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------
// resolveThreadID returns a thread ID, preferring the stopped thread from the
// stop location, otherwise fetching threads and returning the first one.
func (s *DapSession) resolveThreadID(timeout time.Duration) (int, error) {
	s.mu.Lock()
	if s.stop.ThreadID > 0 {
		tid := s.stop.ThreadID
		s.mu.Unlock()
		return tid, nil
	}
	s.mu.Unlock()
	threads, err := s.Threads(timeout)
	if err != nil {
		return 0, fmt.Errorf("resolveThreadID: %w", err)
	}
	if len(threads) == 0 {
		return 0, fmt.Errorf("resolveThreadID: no threads available")
	}
	return threads[0].Id, nil
}

// fetchTopFrame fetches the top stack frame if the session is stopped and
// updates the stop location with frame details.
func (s *DapSession) fetchTopFrame(timeout time.Duration) {
	s.mu.Lock()
	status := s.status
	threadID := s.stop.ThreadID
	s.mu.Unlock()
	if status != DapStatusStopped || threadID <= 0 {
		return
	}
	args := DapStackTraceArguments{
		ThreadId:   threadID,
		Levels:     1,
		StartFrame: 0,
	}
	body, err := s.client.SendRequest("stackTrace", args, timeout)
	if err != nil {
		return
	}
	raw, ok := body.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return
	}
	var resp DapStackTraceResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return
	}
	if len(resp.StackFrames) == 0 {
		return
	}
	frame := resp.StackFrames[0]
	s.mu.Lock()
	s.lastStackFrames = resp.StackFrames
	if frame.Source != nil {
		s.stop.Source = frame.Source
	}
	if frame.Line > 0 {
		s.stop.Line = frame.Line
	}
	if frame.Column > 0 {
		s.stop.Column = frame.Column
	}
	s.mu.Unlock()
}

// step is shared logic for Continue, StepOver, StepIn, and StepOut.
// It resolves the thread ID, sets status to Running, sends the stepping
func (s *DapSession) CompleteLaunch(timeout time.Duration) error {
	s.mu.Lock()
	if s.configurationDoneSent || s.launchErrCh == nil {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()
	if s.needsConfigurationDone {
		args := map[string]any{}
		if _, err := s.client.SendRequest("configurationDone", args, timeout); err != nil {
			return fmt.Errorf("dap: configurationDone: %w", err)
		}
		s.mu.Lock()
		s.configurationDoneSent = true
		s.mu.Unlock()
	}
	// Wait for launch to complete
	if err := <-s.launchErrCh; err != nil {
		return fmt.Errorf("dap: launch: %w", err)
	}
	return nil
}
func (s *DapSession) step(command string, timeout time.Duration) (*DapContinueOutcome, error) {
	threadID, err := s.resolveThreadID(timeout)
	if err != nil {
		return nil, fmt.Errorf("dap: %s: %w", command, err)
	}
	s.mu.Lock()
	s.status = DapStatusRunning
	s.mu.Unlock()
	// Send the command.
	var args interface{}
	switch command {
	case "continue":
		args = DapContinueArguments{ThreadId: threadID}
	case "next", "stepIn", "stepOut":
		args = DapStepArguments{ThreadId: threadID}
	}
	if _, err := s.client.SendRequest(command, args, timeout); err != nil {
		return nil, fmt.Errorf("dap: %s: %w", command, err)
	}
	// Wait for stopped event.
	timedOut := false
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-s.stoppedCh:
	case <-timer.C:
		timedOut = true
	}
	// Update stop location with frame info if available.
	s.fetchTopFrame(timeout)
	summary := s.Summary()
	outcome := &DapContinueOutcome{
		Snapshot: summary,
		TimedOut: timedOut,
	}
	if timedOut {
		outcome.State = "timeout"
	} else {
		outcome.State = string(s.status)
	}
	return outcome, nil
}
