package custom

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/awsl-project/maxx/internal/domain"
	"github.com/tidwall/gjson"
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

	// Should have 1 entry: Claude Code prompt
	if len(system) != 1 {
		t.Fatalf("Expected 1 system entry, got %d", len(system))
	}

	// Check first entry is Claude Code prompt
	entry0, ok := system[0].(map[string]interface{})
	if !ok {
		t.Fatalf("system entry 0 is not a map: %T", system[0])
	}
	if entry0["type"] != "text" {
		t.Errorf("Expected entry 0 type='text', got %v", entry0["type"])
	}
	if entry0["text"] != claudeCodeSystemPrompt {
		t.Errorf("Expected entry 0 text='%s', got %v", claudeCodeSystemPrompt, entry0["text"])
	}
}

func TestUserIDGeneration(t *testing.T) {
	userID := generateFakeUserID()

	// Check format matches expected regex
	if !isValidUserID(userID) {
		t.Errorf("Generated user_id doesn't match expected format: %s", userID)
	}
}

func TestCloakingForNonClaudeClient(t *testing.T) {
	body := []byte(`{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"hello"}]}`)

	// Non-Claude Code client (e.g., curl)
	result := applyCloaking(body, "curl/7.68.0", "claude-3-5-sonnet", nil)

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
}

func TestNoCloakingForClaudeClient(t *testing.T) {
	body := []byte(`{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"hello"}]}`)

	// Claude Code client
	result := applyCloaking(body, "claude-cli/2.1.23 (external, cli)", "claude-3-5-sonnet", nil)

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

func TestShouldCloakModes(t *testing.T) {
	if !shouldCloak("", "curl/7.68.0") {
		t.Error("default mode should cloak non-claude clients")
	}
	if shouldCloak("", "claude-cli/2.1.17 (external, cli)") {
		t.Error("default mode should not cloak claude-cli clients")
	}
	if !shouldCloak("always", "claude-cli/2.1.17 (external, cli)") {
		t.Error("always mode should cloak all clients")
	}
	if shouldCloak("never", "curl/7.68.0") {
		t.Error("never mode should cloak none")
	}
}

func TestSkipSystemInjectionForHaiku(t *testing.T) {
	body := []byte(`{"model":"claude-3-5-haiku-20241022","messages":[{"role":"user","content":"hello"}]}`)

	result := applyCloaking(body, "curl/7.68.0", "claude-3-5-haiku-20241022", nil)

	if gjson.GetBytes(result, "system").Exists() {
		t.Error("system prompt should be skipped for claude-3-5-haiku models")
	}
	if !gjson.GetBytes(result, "metadata.user_id").Exists() {
		t.Error("user_id should still be injected for haiku models")
	}
}

func TestFullBodyProcessingAddsCacheControlAndExtractsBetas(t *testing.T) {
	body := []byte(`{
		"model":"claude-3-5-sonnet",
		"betas":["custom-beta-1"],
		"system":[{"type":"text","text":"You are helpful"}],
		"tools":[{"name":"test_tool","description":"A test tool"}],
		"messages":[
			{"role":"user","content":"hello"},
			{"role":"assistant","content":"ok"},
			{"role":"user","content":"again"}
		]
	}`)

	result, betas := processClaudeRequestBody(body, "curl/7.68.0", nil)

	if len(betas) != 1 || betas[0] != "custom-beta-1" {
		t.Fatalf("expected betas to be extracted, got %v", betas)
	}
	if gjson.GetBytes(result, "betas").Exists() {
		t.Error("betas should be removed from body")
	}

	if !gjson.GetBytes(result, "tools.0.cache_control").Exists() {
		t.Error("cache_control should be injected into tools")
	}
	if gjson.GetBytes(result, "system.0.cache_control").Exists() || gjson.GetBytes(result, "system.1.cache_control").Exists() {
		// ok
	} else {
		t.Error("cache_control should be injected into system")
	}
	if !gjson.GetBytes(result, "messages.0.content.0.cache_control").Exists() {
		t.Error("cache_control should be injected into second-to-last user message")
	}
}

func TestSensitiveWordObfuscation(t *testing.T) {
	body := []byte(`{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"this is secret"}]}`)
	cfg := &domain.ProviderConfigCustomCloak{
		Mode:           "always",
		SensitiveWords: []string{"secret"},
	}

	result := applyCloaking(body, "curl/7.68.0", "claude-3-5-sonnet", cfg)

	const zwsp = "\u200B"
	if strings.Contains(string(result), "secret") {
		t.Error("sensitive word should be obfuscated")
	}
	if !strings.Contains(string(result), "s"+zwsp+"ecret") {
		t.Error("obfuscated word should include zero-width space")
	}
}

func TestStrictCloakingReplacesSystem(t *testing.T) {
	body := []byte(`{
		"model":"claude-3-5-sonnet",
		"system":[
			{"type":"text","text":"Original system"},
			{"type":"text","text":"More system"}
		],
		"messages":[{"role":"user","content":"hello"}]
	}`)
	cfg := &domain.ProviderConfigCustomCloak{
		Mode:       "always",
		StrictMode: true,
	}

	result := applyCloaking(body, "curl/7.68.0", "claude-3-5-sonnet", cfg)

	system := gjson.GetBytes(result, "system")
	if !system.IsArray() || len(system.Array()) != 1 {
		t.Fatalf("strict mode should replace system with single entry, got %s", system.Raw)
	}
	if system.Array()[0].Get("text").String() != claudeCodeSystemPrompt {
		t.Errorf("strict mode system text mismatch: %s", system.Array()[0].Get("text").String())
	}
}

func TestSensitiveWordObfuscationInSystem(t *testing.T) {
	body := []byte(`{
		"model":"claude-3-5-sonnet",
		"system":[{"type":"text","text":"keep secret here"}],
		"messages":[{"role":"user","content":"hello"}]
	}`)
	cfg := &domain.ProviderConfigCustomCloak{
		Mode:           "always",
		SensitiveWords: []string{"secret"},
	}

	result := applyCloaking(body, "curl/7.68.0", "claude-3-5-sonnet", cfg)

	const zwsp = "\u200B"
	if strings.Contains(string(result), "secret") {
		t.Error("sensitive word in system should be obfuscated")
	}
	if !strings.Contains(string(result), "s"+zwsp+"ecret") {
		t.Error("obfuscated system word should include zero-width space")
	}
}

func TestEnsureCacheControlWithSystemString(t *testing.T) {
	body := []byte(`{
		"model":"claude-3-5-sonnet",
		"system":"You are helpful",
		"tools":[{"name":"test_tool","description":"A test tool"}],
		"messages":[
			{"role":"user","content":"hello"},
			{"role":"assistant","content":"ok"},
			{"role":"user","content":"again"}
		]
	}`)
	cfg := &domain.ProviderConfigCustomCloak{Mode: "never"}

	result, _ := processClaudeRequestBody(body, "curl/7.68.0", cfg)

	if !gjson.GetBytes(result, "system.0.cache_control").Exists() {
		t.Error("cache_control should be injected into system string")
	}
	if gjson.GetBytes(result, "system").Type != gjson.JSON {
		t.Error("system should be converted to array when injecting cache_control")
	}
}

func TestEnsureCacheControlDoesNotOverrideExistingTools(t *testing.T) {
	body := []byte(`{
		"model":"claude-3-5-sonnet",
		"tools":[
			{"name":"tool1","cache_control":{"type":"ephemeral"}},
			{"name":"tool2"}
		],
		"messages":[
			{"role":"user","content":"hello"},
			{"role":"assistant","content":"ok"},
			{"role":"user","content":"again"}
		]
	}`)
	cfg := &domain.ProviderConfigCustomCloak{Mode: "never"}

	result, _ := processClaudeRequestBody(body, "curl/7.68.0", cfg)

	if gjson.GetBytes(result, "tools.1.cache_control").Exists() {
		t.Error("cache_control should not be added when tools already have cache_control")
	}
}

func TestDisableThinkingIfToolChoiceForced(t *testing.T) {
	body := []byte(`{
		"model":"claude-3-5-sonnet",
		"thinking":{"type":"enabled","budget_tokens":1000},
		"tool_choice":{"type":"any"}
	}`)

	result := disableThinkingIfToolChoiceForced(body)
	if gjson.GetBytes(result, "thinking").Exists() {
		t.Error("thinking should be removed when tool_choice.type=any")
	}

	bodyAuto := []byte(`{
		"model":"claude-3-5-sonnet",
		"thinking":{"type":"enabled","budget_tokens":1000},
		"tool_choice":{"type":"auto"}
	}`)
	resultAuto := disableThinkingIfToolChoiceForced(bodyAuto)
	if !gjson.GetBytes(resultAuto, "thinking").Exists() {
		t.Error("thinking should remain when tool_choice.type=auto")
	}
}

func TestProcessClaudeRequestBodyDoesNotForceStream(t *testing.T) {
	body := []byte(`{
		"model":"claude-3-5-sonnet",
		"stream":false,
		"messages":[{"role":"user","content":"hello"}]
	}`)
	cfg := &domain.ProviderConfigCustomCloak{Mode: "never"}

	result, _ := processClaudeRequestBody(body, "curl/7.68.0", cfg)
	if gjson.GetBytes(result, "stream").Type != gjson.False {
		t.Error("stream flag should not be forced to true")
	}
}

func TestClaudeToolPrefixApplyAndStrip(t *testing.T) {
	body := []byte(`{
		"model":"claude-3-5-sonnet",
		"tools":[{"name":"t1"},{"type":"web_search","name":"web_search"}],
		"tool_choice":{"type":"tool","name":"t1"},
		"messages":[
			{"role":"assistant","content":[{"type":"tool_use","name":"t1","input":{}}]}
		],
		"content":[{"type":"tool_use","name":"t1"}]
	}`)

	updated := applyClaudeToolPrefix(body, "proxy_")
	if gjson.GetBytes(updated, "tools.0.name").String() != "proxy_t1" {
		t.Error("tool name should be prefixed")
	}
	if gjson.GetBytes(updated, "tools.1.name").String() != "web_search" {
		t.Error("built-in tool name should not be prefixed")
	}
	if gjson.GetBytes(updated, "tool_choice.name").String() != "proxy_t1" {
		t.Error("tool_choice name should be prefixed")
	}
	if gjson.GetBytes(updated, "messages.0.content.0.name").String() != "proxy_t1" {
		t.Error("tool_use name should be prefixed in messages")
	}

	// Simulate response stripping
	responseBody := []byte(`{"content":[{"type":"tool_use","name":"proxy_t1"}]}`)
	stripped := stripClaudeToolPrefixFromResponse(responseBody, "proxy_")
	if gjson.GetBytes(stripped, "content.0.name").String() != "t1" {
		t.Error("tool_use name should be stripped in response content")
	}
}

func TestStripClaudeToolPrefixFromStreamLine(t *testing.T) {
	line := "data: {\"type\":\"content_block_start\",\"content_block\":{\"type\":\"tool_use\",\"name\":\"proxy_t1\"}}\n"
	out := stripClaudeToolPrefixFromStreamLine([]byte(line), "proxy_")
	if !strings.Contains(string(out), "\"name\":\"t1\"") {
		t.Error("stream line tool name should be stripped")
	}
}

func TestNoDuplicateSystemPromptInjection(t *testing.T) {
	// Body that already has Claude Code system prompt
	body := []byte(`{
		"model":"claude-3-5-sonnet",
		"messages":[{"role":"user","content":"hello"}],
		"system":[{"type":"text","text":"You are Claude Code, Anthropic's official CLI for Claude."},{"type":"text","text":"Additional instructions"}]
	}`)

	result := injectClaudeCodeSystemPrompt(body)

	// Count occurrences of "Claude Code"
	count := strings.Count(string(result), "Claude Code")
	if count != 1 {
		t.Errorf("Expected 1 occurrence of 'Claude Code', got %d", count)
	}
}
