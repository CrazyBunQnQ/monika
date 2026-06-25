package dap

import (
	"fmt"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// DapManager
// ---------------------------------------------------------------------------

// DapManager is the global DAP session orchestrator. It manages the lifecycle
// of debug sessions and pushes state changes to registered callbacks.
type DapManager struct {
	mu              sync.Mutex
	sessions        map[string]*DapSession
	activeSessionID string
	nextID          int
	projectDir      string

	// Event callbacks for pushing state to frontend and LLM context.
	onSessionCreated    func(DapSessionSummary)
	onSessionTerminated func(DapSessionSummary)
	onStateChanged      func(DapSessionSummary)
	onStopped           func(DapSessionSummary)
	onContinued         func(DapSessionSummary)
	onOutput            func(sessionID string, output string)
}

// NewDapManager creates a new DapManager with the given project directory.
func NewDapManager(projectDir string) *DapManager {
	return &DapManager{
		sessions:   make(map[string]*DapSession),
		projectDir: projectDir,
		nextID:     1,
	}
}

// SetProjectDir updates the default project directory (used as cwd fallback).
func (m *DapManager) SetProjectDir(dir string) {
	m.mu.Lock()
	m.projectDir = dir
	m.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Callback setters
// ---------------------------------------------------------------------------

func (m *DapManager) OnSessionCreated(fn func(DapSessionSummary)) {
	m.mu.Lock()
	m.onSessionCreated = fn
	m.mu.Unlock()
}

func (m *DapManager) OnSessionTerminated(fn func(DapSessionSummary)) {
	m.mu.Lock()
	m.onSessionTerminated = fn
	m.mu.Unlock()
}

func (m *DapManager) OnStateChanged(fn func(DapSessionSummary)) {
	m.mu.Lock()
	m.onStateChanged = fn
	m.mu.Unlock()
}

func (m *DapManager) OnStopped(fn func(DapSessionSummary)) {
	m.mu.Lock()
	m.onStopped = fn
	m.mu.Unlock()
}

func (m *DapManager) OnContinued(fn func(DapSessionSummary)) {
	m.mu.Lock()
	m.onContinued = fn
	m.mu.Unlock()
}

func (m *DapManager) OnOutput(fn func(sessionID string, output string)) {
	m.mu.Lock()
	m.onOutput = fn
	m.mu.Unlock()
}

// ---------------------------------------------------------------------------
// ID generation
// ---------------------------------------------------------------------------

func (m *DapManager) nextSessionID() string {
	id := m.nextID
	m.nextID++
	return fmt.Sprintf("session-%d", id)
}

// ---------------------------------------------------------------------------
// launch
// ---------------------------------------------------------------------------

// Launch starts a new debug session by launching a program.
func (m *DapManager) Launch(program string, args []string, adapterName string, cwd string) (*DapSessionSummary, error) {
	if cwd == "" {
		cwd = m.projectDir
	}

	adapter := selectLaunchAdapter(program, cwd, adapterName)
	if adapter == nil {
		if adapterName != "" {
			return nil, fmt.Errorf("dap: adapter %q not found for program %q", adapterName, program)
		}
		return nil, fmt.Errorf("dap: no suitable adapter found for program %q", program)
	}

	m.mu.Lock()
	if m.activeSessionID != "" {
		if s, ok := m.sessions[m.activeSessionID]; ok && s != nil {
			summary := s.Summary()
			if summary.Status != DapStatusTerminated {
				m.mu.Unlock()
				return nil, fmt.Errorf("dap: active session %q is still %s; terminate it first", m.activeSessionID, summary.Status)
			}
		}
	}
	m.mu.Unlock()

	client, err := SpawnDapClient(adapter, cwd)
	if err != nil {
		return nil, fmt.Errorf("dap: spawn client: %w", err)
	}

	sessionID := m.nextSessionID()
	session := newDapSession(sessionID, client, adapter, cwd, program)
	m.setupSessionEvents(session)

	// Send initialize request.
	initArgs := DapInitializeArguments{
		ClientID:        "monika",
		ClientName:      "Monika",
		AdapterID:       adapter.Name,
		PathFormat:      "path",
		LinesStartAt1:   true,
		ColumnsStartAt1: true,
	}
	caps, err := client.Initialize(initArgs, DefaultRequestTimeout)
	if err != nil {
		client.Dispose()
		return nil, fmt.Errorf("dap: initialize: %w", err)
	}

	// Send launch request in background (some adapters like debugpy require
	// configurationDone to be sent during the launch request handling).
	launchArgs := DapLaunchArguments{
		Program: program,
		Args:    args,
		Cwd:     cwd,
	}
	launchErrCh := make(chan error, 1)
	go func() {
		_, err := client.SendRequest("launch", launchArgs, DefaultRequestTimeout)
		launchErrCh <- err
	}()
	time.Sleep(200 * time.Millisecond)

	// Store the launchErrCh on the session so CompleteLaunch can wait for it.
	session.launchErrCh = launchErrCh
	session.capabilities = caps
	session.needsConfigurationDone = caps.SupportsConfigurationDoneRequest

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.activeSessionID = sessionID
	m.mu.Unlock()

	summary := session.Summary()
	if fn := m.onSessionCreated; fn != nil {
		fn(summary)
	}
	return &summary, nil
}

// ---------------------------------------------------------------------------
// attach
// ---------------------------------------------------------------------------

// Attach starts a new debug session by attaching to an existing process.
func (m *DapManager) Attach(pid int, port int, host string, adapterName string, cwd string) (*DapSessionSummary, error) {
	if cwd == "" {
		cwd = m.projectDir
	}

	adapter := selectAttachAdapter(cwd, adapterName)
	if adapter == nil {
		if adapterName != "" {
			return nil, fmt.Errorf("dap: adapter %q not found for attach", adapterName)
		}
		return nil, fmt.Errorf("dap: no suitable adapter found for attach in %q", cwd)
	}

	m.mu.Lock()
	if m.activeSessionID != "" {
		if s, ok := m.sessions[m.activeSessionID]; ok && s != nil {
			summary := s.Summary()
			if summary.Status != DapStatusTerminated {
				m.mu.Unlock()
				return nil, fmt.Errorf("dap: active session %q is still %s; terminate it first", m.activeSessionID, summary.Status)
			}
		}
	}
	m.mu.Unlock()

	client, err := SpawnDapClient(adapter, cwd)
	if err != nil {
		return nil, fmt.Errorf("dap: spawn client: %w", err)
	}

	program := fmt.Sprintf("pid-%d", pid)
	sessionID := m.nextSessionID()
	session := newDapSession(sessionID, client, adapter, cwd, program)
	m.setupSessionEvents(session)

	// Send initialize request.
	initArgs := DapInitializeArguments{
		ClientID:        "monika",
		ClientName:      "Monika",
		AdapterID:       adapter.Name,
		PathFormat:      "path",
		LinesStartAt1:   true,
		ColumnsStartAt1: true,
	}
	_, err = client.Initialize(initArgs, DefaultRequestTimeout)
	if err != nil {
		client.Dispose()
		return nil, fmt.Errorf("dap: initialize: %w", err)
	}

	// Send attach request.
	attachArgs := DapAttachArguments{
		PID:  pid,
		Port: port,
		Host: host,
		Cwd:  cwd,
	}
	_, err = client.SendRequest("attach", attachArgs, DefaultRequestTimeout)
	if err != nil {
		client.Dispose()
		return nil, fmt.Errorf("dap: attach: %w", err)
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.activeSessionID = sessionID
	m.mu.Unlock()

	summary := session.Summary()
	if fn := m.onSessionCreated; fn != nil {
		fn(summary)
	}
	return &summary, nil
}

// ---------------------------------------------------------------------------
// Event wiring (manager-level callbacks)
// ---------------------------------------------------------------------------

// setupSessionEvents registers manager-level event handlers on the session's
// client. These fire the callbacks set via OnStateChanged / OnOutput /
// OnSessionTerminated.
func (m *DapManager) setupSessionEvents(session *DapSession) {
	client := session.client

	client.OnEvent(DapEventStopped, func(_ interface{}, _ *DapEventMessage) {
		summary := session.Summary()
		if fn := m.onStopped; fn != nil {
			fn(summary)
		}
	})

	client.OnEvent(DapEventContinued, func(_ interface{}, _ *DapEventMessage) {
		summary := session.Summary()
		if fn := m.onContinued; fn != nil {
			fn(summary)
		}
	})

	client.OnEvent(DapEventOutput, func(_ interface{}, _ *DapEventMessage) {
		output := session.GetOutput()
		if fn := m.onOutput; fn != nil {
			fn(session.Summary().ID, output)
		}
	})

	client.OnEvent(DapEventExited, func(_ interface{}, _ *DapEventMessage) {
		summary := session.Summary()
		if fn := m.onStateChanged; fn != nil {
			fn(summary)
		}
	})

	client.OnEvent(DapEventTerminated, func(_ interface{}, _ *DapEventMessage) {
		summary := session.Summary()
		if fn := m.onStateChanged; fn != nil {
			fn(summary)
		}
		if fn := m.onSessionTerminated; fn != nil {
			fn(summary)
		}
	})
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

// GetSession returns the session with the given ID, or nil.
func (m *DapManager) GetSession(id string) *DapSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[id]
}

// ActiveSession returns the active session, or the most recently used
// non-terminated session if there is no explicit active session.
func (m *DapManager) ActiveSession() *DapSession {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Prefer the explicitly active session if it exists and is not terminated.
	if m.activeSessionID != "" {
		if s, ok := m.sessions[m.activeSessionID]; ok && s != nil {
			summary := s.Summary()
			if summary.Status != DapStatusTerminated {
				return s
			}
		}
	}

	// Fall back to the most recently created non-terminated session.
	var best *DapSession
	for _, s := range m.sessions {
		summary := s.Summary()
		if summary.Status == DapStatusTerminated {
			continue
		}
		best = s
	}
	return best
}

// ListSessions returns a summary snapshot for every managed session.
func (m *DapManager) ListSessions() []DapSessionSummary {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]DapSessionSummary, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s.Summary())
	}
	return out
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// TerminateSession terminates and removes the session with the given ID.
func (m *DapManager) TerminateSession(id string) {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	delete(m.sessions, id)
	if m.activeSessionID == id {
		m.activeSessionID = ""
	}
	m.mu.Unlock()

	s.Terminate(DefaultRequestTimeout)

	summary := s.Summary()
	if fn := m.onStateChanged; fn != nil {
		fn(summary)
	}
	if fn := m.onSessionTerminated; fn != nil {
		fn(summary)
	}
}

// TerminateAll terminates and removes all managed sessions.
func (m *DapManager) TerminateAll() {
	m.mu.Lock()
	sessions := make([]*DapSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.sessions = make(map[string]*DapSession)
	m.activeSessionID = ""
	m.mu.Unlock()

	for _, s := range sessions {
		s.Terminate(DefaultRequestTimeout)
		summary := s.Summary()
		if fn := m.onStateChanged; fn != nil {
			fn(summary)
		}
		if fn := m.onSessionTerminated; fn != nil {
			fn(summary)
		}
	}
}
