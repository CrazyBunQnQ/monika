package dap

import (
	"encoding/json"
	"testing"
)

func TestTypesJSONRoundtrip(t *testing.T) {
	msg := DapRequestMessage{
		DapProtocolMessage: DapProtocolMessage{
			Seq:  1,
			Type: "request",
		},
		Command:   "initialize",
		Arguments: json.RawMessage(`{"clientID":"monika"}`),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded DapRequestMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Seq != 1 || decoded.Command != "initialize" {
		t.Fatalf("roundtrip mismatch: %+v", decoded)
	}
}

func TestCapabilitiesOmitempty(t *testing.T) {
	caps := DapCapabilities{}
	data, err := json.Marshal(caps)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{}" {
		t.Fatalf("expected empty object, got: %s", data)
	}
}

func TestDefaultAdapters(t *testing.T) {
	if len(DefaultAdapters) != 5 {
		t.Fatalf("expected 5 adapters, got %d", len(DefaultAdapters))
	}
	for name, cfg := range DefaultAdapters {
		if cfg.Command == "" {
			t.Errorf("adapter %q has empty command", name)
		}
		if len(cfg.FileTypes) == 0 {
			t.Errorf("adapter %q has no file types", name)
		}
		if cfg.ConnectMode != "stdio" {
			t.Errorf("adapter %q expected stdio, got %q", name, cfg.ConnectMode)
		}
	}
}

func TestGetAvailableAdapters(t *testing.T) {
	adapters := GetAvailableAdapters(t.TempDir())
	t.Logf("found %d available adapters", len(adapters))
	for _, a := range adapters {
		t.Logf("  - %s at %s", a.Name, a.ResolvedCommand)
	}
}

func TestNewDapManager(t *testing.T) {
	mgr := NewDapManager(t.TempDir())
	if mgr == nil {
		t.Fatal("NewDapManager returned nil")
	}
	sessions := mgr.ListSessions()
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestDapPendingRequestJSONTags(t *testing.T) {
	data, err := json.Marshal(DapPendingRequest{
		Command: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["command"] != "test" {
		t.Errorf("expected command=test, got %v", decoded["command"])
	}
	if _, ok := decoded["Resolve"]; ok {
		t.Error("Resolve should not be serialized")
	}
	if _, ok := decoded["Reject"]; ok {
		t.Error("Reject should not be serialized")
	}
}

func TestSessionStatusConstants(t *testing.T) {
	if DapStatusLaunching != "launching" ||
		DapStatusConfiguring != "configuring" ||
		DapStatusStopped != "stopped" ||
		DapStatusRunning != "running" ||
		DapStatusTerminated != "terminated" {
		t.Error("unexpected session status constants")
	}
}

func TestEventTypeConstants(t *testing.T) {
	if DapEventStopped != "stopped" ||
		DapEventContinued != "continued" ||
		DapEventOutput != "output" ||
		DapEventExited != "exited" ||
		DapEventTerminated != "terminated" ||
		DapEventInitialized != "initialized" {
		t.Error("unexpected event type constants")
	}
}

func TestSessionSummaryConstruction(t *testing.T) {
	session := newDapSession("test-1", nil, nil, "/tmp", "main.go")
	summary := session.Summary()
	if summary.ID != "test-1" {
		t.Errorf("expected id test-1, got %s", summary.ID)
	}
	if summary.Status != "launching" {
		t.Errorf("expected status launching, got %s", summary.Status)
	}
	if summary.Adapter != "" {
		t.Errorf("expected empty adapter, got %s", summary.Adapter)
	}
	if !summary.IsLocal {
		t.Error("IsLocal should be true (default for DAP local sessions)")
	}
}
