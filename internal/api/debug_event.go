package api

import (
	"encoding/json"
	"monika/internal/dap"
)

const (
	DebugSessionCreated    = "debug.session.created"
	DebugSessionTerminated = "debug.session.terminated"
	DebugStopped           = "debug.stopped"
	DebugContinued         = "debug.continued"
	DebugOutput            = "debug.output"
	DebugStateChanged      = "debug.state.changed"
)

// EmitDebugEvent emits a DAP event through the EventBus.
func (eb *EventBus) EmitDebugEvent(eventType string, summary dap.DapSessionSummary) {
	data, _ := json.Marshal(summary)
	eb.Emit(StreamEvent{
		Type:    eventType,
		Content: string(data),
	})
}

// EmitDebugOutput emits debug output through the EventBus.
func (eb *EventBus) EmitDebugOutput(sessionID string, output string) {
	eb.Emit(StreamEvent{
		Type:    DebugOutput,
		Content: output,
	})
}
