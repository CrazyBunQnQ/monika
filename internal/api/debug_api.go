package api

import (
	"fmt"
	"monika/internal/dap"
)

// DebugAPI provides Wails bindings for frontend debug operations.
type DebugAPI struct {
	manager *dap.DapManager
}

func NewDebugAPI(manager *dap.DapManager) *DebugAPI {
	return &DebugAPI{manager: manager}
}

func (api *DebugAPI) Launch(program string, args []string, adapter string, cwd string) (*dap.DapSessionSummary, error) {
	return api.manager.Launch(program, args, adapter, cwd)
}

func (api *DebugAPI) Attach(pid int, port int, host string, adapter string, cwd string) (*dap.DapSessionSummary, error) {
	return api.manager.Attach(pid, port, host, adapter, cwd)
}

func (api *DebugAPI) Stop(sessionID string) {
	api.manager.TerminateSession(sessionID)
}

func (api *DebugAPI) Continue(sessionID string) (*dap.DapContinueOutcome, error) {
	session := api.resolveSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("no active debug session")
	}
	return session.Continue(dap.DefaultRequestTimeout)
}

func (api *DebugAPI) StepOver(sessionID string) (*dap.DapContinueOutcome, error) {
	session := api.resolveSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("no active debug session")
	}
	return session.StepOver(dap.DefaultRequestTimeout)
}

func (api *DebugAPI) StepIn(sessionID string) (*dap.DapContinueOutcome, error) {
	session := api.resolveSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("no active debug session")
	}
	return session.StepIn(dap.DefaultRequestTimeout)
}

func (api *DebugAPI) StepOut(sessionID string) (*dap.DapContinueOutcome, error) {
	session := api.resolveSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("no active debug session")
	}
	return session.StepOut(dap.DefaultRequestTimeout)
}

func (api *DebugAPI) Pause(sessionID string) (*dap.DapSessionSummary, error) {
	session := api.resolveSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("no active debug session")
	}
	return session.Pause(dap.DefaultRequestTimeout)
}

func (api *DebugAPI) GetState(sessionID string) (*dap.DapSessionSummary, error) {
	session := api.resolveSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("no active debug session")
	}
	s := session.Summary()
	return &s, nil
}

func (api *DebugAPI) ListSessions() []dap.DapSessionSummary {
	return api.manager.ListSessions()
}

func (api *DebugAPI) GetVariables(sessionID string, variablesRef int) ([]dap.DapVariable, error) {
	session := api.resolveSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("no active debug session")
	}
	return session.Variables(variablesRef, dap.DefaultRequestTimeout)
}

func (api *DebugAPI) SetBreakpoint(sessionID string, file string, line int, condition string) ([]dap.DapBreakpointRecord, error) {
	session := api.resolveSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("no active debug session")
	}
	return session.SetBreakpoint(file, line, condition, dap.DefaultRequestTimeout)
}

func (api *DebugAPI) RemoveBreakpoint(sessionID string, file string, line int) ([]dap.DapBreakpointRecord, error) {
	session := api.resolveSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("no active debug session")
	}
	return session.RemoveBreakpoint(file, line, dap.DefaultRequestTimeout)
}

func (api *DebugAPI) GetScopes(sessionID string, frameID int) ([]dap.DapScope, error) {
	session := api.resolveSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("no active debug session")
	}
	return session.Scopes(frameID, dap.DefaultRequestTimeout)
}

func (api *DebugAPI) GetStackTrace(sessionID string, levels int) ([]dap.DapStackFrame, error) {
	session := api.resolveSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("no active debug session")
	}
	return session.StackTrace(levels, dap.DefaultRequestTimeout)
}

func (api *DebugAPI) GetThreads(sessionID string) ([]dap.DapThread, error) {
	session := api.resolveSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("no active debug session")
	}
	return session.Threads(dap.DefaultRequestTimeout)
}

func (api *DebugAPI) GetKeys(sessionID string, scopesID int) ([]dap.DapScope, error) {
	session := api.resolveSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("no active debug session")
	}
	return session.Scopes(scopesID, dap.DefaultRequestTimeout)
}

func (api *DebugAPI) GetOutput(sessionID string) (string, error) {
	session := api.resolveSession(sessionID)
	if session == nil {
		return "", fmt.Errorf("no active debug session")
	}
	return session.GetOutput(), nil
}

func (api *DebugAPI) resolveSession(sessionID string) *dap.DapSession {
	if sessionID != "" {
		if s := api.manager.GetSession(sessionID); s != nil {
			return s
		}
	}
	return api.manager.ActiveSession()
}
