package converter

import "testing"

func TestCleanJSONSchema(t *testing.T) {
	schema := map[string]interface{}{
		"$schema": "x",
		"type":    []interface{}{"null", "string"},
		"properties": map[string]interface{}{
			"foo": map[string]interface{}{
				"type":    "number",
				"minimum": 1,
			},
		},
	}
	cleanJSONSchema(schema)
	if _, ok := schema["$schema"]; ok {
		t.Fatalf("expected $schema removed")
	}
	if schema["type"] != "string" {
		t.Fatalf("expected type string, got %#v", schema["type"])
	}
	props := schema["properties"].(map[string]interface{})
	if _, ok := props["foo"].(map[string]interface{})["minimum"]; ok {
		t.Fatalf("expected minimum removed")
	}
}

func TestDeepCleanUndefined(t *testing.T) {
	data := map[string]interface{}{
		"a": "[undefined]",
		"b": map[string]interface{}{
			"c": "[undefined]",
		},
	}
	deepCleanUndefined(data)
	if _, ok := data["a"]; ok {
		t.Fatalf("expected a removed")
	}
	if _, ok := data["b"].(map[string]interface{})["c"]; ok {
		t.Fatalf("expected c removed")
	}
}

func TestFilterInvalidThinkingBlocks(t *testing.T) {
	msgs := []ClaudeMessage{{
		Role: "assistant",
		Content: []interface{}{
			map[string]interface{}{"type": "thinking", "thinking": "t", "signature": "short"},
			map[string]interface{}{"type": "text", "text": "hello"},
		},
	}}
	FilterInvalidThinkingBlocks(msgs)
	blocks := msgs[0].Content.([]interface{})
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].(map[string]interface{})["type"] != "text" {
		t.Fatalf("expected invalid thinking converted to text")
	}
}

func TestRemoveTrailingUnsignedThinking(t *testing.T) {
	msgs := []ClaudeMessage{{
		Role: "assistant",
		Content: []interface{}{
			map[string]interface{}{"type": "text", "text": "hi"},
			map[string]interface{}{"type": "thinking", "thinking": "t", "signature": "short"},
		},
	}}
	RemoveTrailingUnsignedThinking(msgs)
	blocks := msgs[0].Content.([]interface{})
	if len(blocks) != 1 {
		t.Fatalf("expected trailing thinking removed")
	}
}

func TestHelperFlags(t *testing.T) {
	if !shouldEnableThinkingByDefault("claude-opus-4-5-thinking") {
		t.Fatalf("expected thinking enabled")
	}
	if targetModelSupportsThinking("gpt-4o") {
		t.Fatalf("expected no thinking support")
	}
	tools := []ClaudeTool{{Type: "web_search_20250305"}}
	if !hasWebSearchTool(tools) {
		t.Fatalf("expected web search tool")
	}
	msgs := []ClaudeMessage{{
		Role: "assistant",
		Content: []interface{}{
			map[string]interface{}{"type": "tool_use"},
		},
	}}
	if !hasFunctionCalls(msgs) {
		t.Fatalf("expected function calls")
	}
	if hasThinkingHistory(msgs) {
		t.Fatalf("expected no thinking history")
	}
	if !shouldDisableThinkingDueToHistory(msgs) {
		t.Fatalf("expected disable due to tool_use without thinking")
	}
}
