package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"monika/internal/agent"
)

func TestAgentManagement_ListCreateDelete(t *testing.T) {
	registry := agent.NewAgentRegistry([]agent.Agent{
		{Name: "general", Source: "builtin"},
		{Name: "explore", Source: "builtin"},
	})

	saveFn := func(args json.RawMessage) error {
		var entry struct {
			Name         string `json:"name"`
			SystemPrompt string `json:"systemPrompt"`
			Description  string `json:"description"`
			Model        string `json:"model"`
		}
		if err := json.Unmarshal(args, &entry); err != nil {
			return err
		}
		registry.Add(agent.Agent{
			Name:         entry.Name,
			SystemPrompt: entry.SystemPrompt,
			Description:  entry.Description,
			Model:        entry.Model,
			IsCustom:     true,
			Source:       "custom",
		})
		return nil
	}
	deleteFn := func(args json.RawMessage) error {
		var req struct{ Name string }
		json.Unmarshal(args, &req)
		a, ok := registry.Get(req.Name)
		if ok {
			a.Disabled = true
			registry.Add(a)
		}
		return nil
	}
	listFn := func() []AgentInfo {
		agents := registry.GetAll()
		result := make([]AgentInfo, len(agents))
		for i, a := range agents {
			result[i] = AgentInfo{
				Name: a.Name, IsCustom: a.IsCustom, Source: a.Source,
				Disabled: a.Disabled, Model: a.Model,
			}
		}
		return result
	}
	checkFn := func(name string) (bool, bool) {
		a, ok := registry.Get(name)
		if !ok {
			return false, false
		}
		return a.IsCustom, true
	}

	listTool := NewAgentListTool(listFn)
	createTool := NewAgentCreateTool(saveFn)
	deleteTool := NewAgentDeleteTool(deleteFn, checkFn)

	// 1. List initial — should have 2 builtin agents
	result, err := listTool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Content, "general") || !strings.Contains(result.Content, "explore") {
		t.Fatalf("expected general and explore in list, got: %s", result.Content)
	}
	t.Logf("[1] Initial list OK:\n%s", result.Content)

	// 2. Create a custom agent
	createArgs, _ := json.Marshal(map[string]string{
		"name":          "code_reviewer",
		"system_prompt": "You are a code reviewer.",
		"description":   "Reviews code for bugs and style issues.",
		"model":         "deepseek/deepseek-chat",
	})
	result, err = createTool.Execute(context.Background(), createArgs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Content, "code_reviewer") {
		t.Fatalf("expected success message with agent name, got: %s", result.Content)
	}
	t.Logf("[2] Create OK:\n%s", result.Content)

	// 3. List again — should now include code_reviewer
	result, err = listTool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Content, "code_reviewer") {
		t.Fatalf("expected code_reviewer in list after create, got: %s", result.Content)
	}
	t.Logf("[3] List after create OK:\n%s", result.Content)

	// 4. Try to delete a builtin agent — should fail
	delArgs, _ := json.Marshal(map[string]string{"name": "general"})
	result, err = deleteTool.Execute(context.Background(), delArgs)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error when deleting builtin agent")
	}
	t.Logf("[4] Builtin protection OK:\n%s", result.Content)

	// 5. Delete the custom agent — should succeed
	delArgs, _ = json.Marshal(map[string]string{"name": "code_reviewer"})
	result, err = deleteTool.Execute(context.Background(), delArgs)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success deleting custom agent, got: %s", result.Content)
	}
	t.Logf("[5] Delete custom OK:\n%s", result.Content)

	// 6. List after delete — code_reviewer should show [disabled]
	result, err = listTool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Content, "disabled") {
		t.Fatalf("expected [disabled] tag in list, got: %s", result.Content)
	}
	t.Logf("[6] List after delete OK:\n%s", result.Content)
}

func TestAgentManagement_CreateValidation(t *testing.T) {
	createTool := NewAgentCreateTool(func(args json.RawMessage) error { return nil })

	// Missing name
	args, _ := json.Marshal(map[string]string{"system_prompt": "test"})
	result, err := createTool.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing name")
	}
	t.Logf("Missing name rejected: %s", result.Content)

	// Missing system_prompt
	args, _ = json.Marshal(map[string]string{"name": "test_agent"})
	result, err = createTool.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing system_prompt")
	}
	t.Logf("Missing system_prompt rejected: %s", result.Content)
}

func TestAgentManagement_DeleteNonExistent(t *testing.T) {
	registry := agent.NewAgentRegistry(nil)
	deleteTool := NewAgentDeleteTool(
		func(args json.RawMessage) error { return nil },
		func(name string) (bool, bool) {
			_, ok := registry.Get(name)
			return false, ok
		},
	)
	args, _ := json.Marshal(map[string]string{"name": "ghost"})
	result, err := deleteTool.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for non-existent agent")
	}
	t.Logf("Non-existent agent rejected: %s", result.Content)
}
