package custom

import (
	"encoding/json"
	"testing"
)

func TestSystemPromptInjection(t *testing.T) {
	// Test case: empty body
	body := []byte(`{"model":"claude-3-5-sonnet","messages":[]}`)
	result := injectClaudeCodeSystemPrompt(body)

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Check system field exists and is array
	system, ok := parsed["system"].([]interface{})
	if !ok {
		t.Fatalf("system field is not an array: %T", parsed["system"])
	}

	if len(system) == 0 {
		t.Fatal("system array is empty")
	}

	// Check first entry has correct format
	entry, ok := system[0].(map[string]interface{})
	if !ok {
		t.Fatalf("system entry is not a map: %T", system[0])
	}

	if entry["type"] != "text" {
		t.Errorf("Expected type='text', got %v", entry["type"])
	}

	if entry["text"] != claudeCodeSystemPrompt {
		t.Errorf("Expected text='%s', got %v", claudeCodeSystemPrompt, entry["text"])
	}

	t.Logf("Injected system prompt: %s", string(result))
}

func TestUserIDGeneration(t *testing.T) {
	userID := generateFakeUserID()

	// Check format matches sub2api's regex: ^user_[a-fA-F0-9]{64}_account__session_[\w-]+$
	if !isValidUserID(userID) {
		t.Errorf("Generated user_id doesn't match expected format: %s", userID)
	}

	t.Logf("Generated user_id: %s", userID)
}

func TestCloakingForNonClaudeClient(t *testing.T) {
	body := []byte(`{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"hello"}]}`)

	// Non-Claude Code client (e.g., curl)
	result := applyCloaking(body, "curl/7.68.0")

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Should have system prompt injected
	system, ok := parsed["system"].([]interface{})
	if !ok || len(system) == 0 {
		t.Error("System prompt was not injected for non-Claude client")
	}

	// Should have metadata.user_id injected
	metadata, ok := parsed["metadata"].(map[string]interface{})
	if !ok {
		t.Error("metadata was not created")
	}

	userID, ok := metadata["user_id"].(string)
	if !ok || userID == "" {
		t.Error("user_id was not injected")
	}

	if !isValidUserID(userID) {
		t.Errorf("Injected user_id doesn't match expected format: %s", userID)
	}

	t.Logf("Cloaked body: %s", string(result))
}

func TestNoCloakingForClaudeClient(t *testing.T) {
	body := []byte(`{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"hello"}]}`)

	// Claude Code client
	result := applyCloaking(body, "claude-cli/2.1.17 (external, cli)")

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Should NOT have system prompt injected
	if _, ok := parsed["system"]; ok {
		t.Error("System prompt was injected for Claude Code client (should not)")
	}

	// Should NOT have metadata injected
	if _, ok := parsed["metadata"]; ok {
		t.Error("metadata was injected for Claude Code client (should not)")
	}
}
