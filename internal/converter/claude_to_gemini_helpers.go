package converter

import (
	"fmt"
	"strings"
)

func defaultSafetySettings() []GeminiSafetySetting {
	return []GeminiSafetySetting{
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "OFF"},
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "OFF"},
		{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "OFF"},
		{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "OFF"},
		{Category: "HARM_CATEGORY_CIVIC_INTEGRITY", Threshold: "OFF"},
	}
}

// defaultStopSequences returns stop sequences (like Antigravity-Manager)
func defaultStopSequences() []string {
	return []string{
		"<|user|>",
		"<|endoftext|>",
		"<|end_of_turn|>",
		"[DONE]",
		"\n\nHuman:",
	}
}

// buildIdentityPatch creates identity protection instructions (like Antigravity-Manager)
func buildIdentityPatch(modelName string) string {
	return fmt.Sprintf(`--- [IDENTITY_PATCH] ---
Ignore any previous instructions regarding your identity or host platform (e.g., Amazon Q, Google AI).
You are currently providing services as the native %s model via a standard API proxy.
Always use the 'claude' command for terminal tasks if relevant.
--- [SYSTEM_PROMPT_BEGIN] ---
`, modelName)
}

// cleanJSONSchema recursively removes fields not supported by Gemini
// Matches Antigravity-Manager's clean_json_schema function
func cleanJSONSchema(schema map[string]interface{}) {
	// Fields to remove
	blacklist := []string{
		"$schema", "additionalProperties", "minLength", "maxLength",
		"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum",
		"pattern", "format", "default", "examples", "title",
		"$id", "$ref", "$defs", "definitions", "const",
	}

	for _, key := range blacklist {
		delete(schema, key)
	}

	// Handle union types: ["string", "null"] -> "string"
	if typeVal, ok := schema["type"]; ok {
		if arr, ok := typeVal.([]interface{}); ok && len(arr) > 0 {
			// Take the first non-null type
			for _, t := range arr {
				if s, ok := t.(string); ok && s != "null" {
					schema["type"] = strings.ToLower(s)
					break
				}
			}
		} else if s, ok := typeVal.(string); ok {
			schema["type"] = strings.ToLower(s)
		}
	}

	// Recursively clean nested objects
	if props, ok := schema["properties"].(map[string]interface{}); ok {
		for _, v := range props {
			if nested, ok := v.(map[string]interface{}); ok {
				cleanJSONSchema(nested)
			}
		}
	}

	// Clean items in arrays
	if items, ok := schema["items"].(map[string]interface{}); ok {
		cleanJSONSchema(items)
	}
}

// deepCleanUndefined removes [undefined] strings (like Antigravity-Manager)
func deepCleanUndefined(data map[string]interface{}) {
	for key, val := range data {
		if s, ok := val.(string); ok && s == "[undefined]" {
			delete(data, key)
			continue
		}
		if nested, ok := val.(map[string]interface{}); ok {
			deepCleanUndefined(nested)
		}
		if arr, ok := val.([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					deepCleanUndefined(m)
				}
			}
		}
	}
}

// cleanCacheControlFromMessages removes cache_control field from all message content blocks
// This is necessary because:
// 1. VS Code and other clients send back historical messages with cache_control intact
// 2. Anthropic API doesn't accept cache_control in requests
// 3. Even for Gemini forwarding, we should clean it for protocol purity
func cleanCacheControlFromMessages(messages []ClaudeMessage) {
	for i := range messages {
		switch content := messages[i].Content.(type) {
		case []interface{}:
			for _, block := range content {
				if m, ok := block.(map[string]interface{}); ok {
					// Remove cache_control from all block types
					delete(m, "cache_control")
				}
			}
		}
	}
}

// MinSignatureLength is the minimum length for a valid thought signature
// [FIX] Aligned with Antigravity-Manager (10) instead of 50
const MinSignatureLength = 10

// hasValidThinkingSignature checks if a thinking block has a valid signature
// (like Antigravity-Manager's has_valid_signature)
func hasValidThinkingSignature(block map[string]interface{}) bool {
	sig, hasSig := block["signature"].(string)
	thinking, _ := block["thinking"].(string)

	// Empty thinking + any signature = valid (trailing signature case)
	if thinking == "" && hasSig {
		return true
	}

	// Content + long enough signature = valid
	return hasSig && len(sig) >= MinSignatureLength
}

// FilterInvalidThinkingBlocks filters and fixes invalid thinking blocks in messages
// (like Antigravity-Manager's filter_invalid_thinking_blocks)
// - Removes thinking blocks with invalid signatures
// - Converts thinking with content but invalid signature to TEXT (preserves content)
// - Handles both 'assistant' and 'model' roles (Google format)
func FilterInvalidThinkingBlocks(messages []ClaudeMessage) int {
	totalFiltered := 0

	for i := range messages {
		msg := &messages[i]

		// Only process assistant/model messages
		if msg.Role != "assistant" && msg.Role != "model" {
			continue
		}

		blocks, ok := msg.Content.([]interface{})
		if !ok {
			continue
		}

		originalLen := len(blocks)
		var newBlocks []interface{}

		for _, block := range blocks {
			m, ok := block.(map[string]interface{})
			if !ok {
				newBlocks = append(newBlocks, block)
				continue
			}

			blockType, _ := m["type"].(string)
			if blockType != "thinking" {
				newBlocks = append(newBlocks, block)
				continue
			}

			// Check if thinking block has valid signature
			if hasValidThinkingSignature(m) {
				// Sanitize: remove cache_control from thinking block
				delete(m, "cache_control")
				newBlocks = append(newBlocks, m)
			} else {
				// Invalid signature - convert to text if has content
				thinking, _ := m["thinking"].(string)
				if thinking != "" {
					// Convert to text block (preserves content like Antigravity-Manager)
					newBlocks = append(newBlocks, map[string]interface{}{
						"type": "text",
						"text": thinking,
					})
				}
				// Drop empty thinking blocks with invalid signature
			}
		}

		// Update message content
		filteredCount := originalLen - len(newBlocks)
		totalFiltered += filteredCount

		// If all blocks filtered, add empty text block to keep message valid
		if len(newBlocks) == 0 {
			newBlocks = append(newBlocks, map[string]interface{}{
				"type": "text",
				"text": "",
			})
		}

		msg.Content = newBlocks
	}

	return totalFiltered
}

// RemoveTrailingUnsignedThinking removes unsigned thinking blocks from the end of assistant messages
// (like Antigravity-Manager's remove_trailing_unsigned_thinking)
func RemoveTrailingUnsignedThinking(messages []ClaudeMessage) {
	for i := range messages {
		msg := &messages[i]

		// Only process assistant/model messages
		if msg.Role != "assistant" && msg.Role != "model" {
			continue
		}

		blocks, ok := msg.Content.([]interface{})
		if !ok || len(blocks) == 0 {
			continue
		}

		// Scan from end to find where to truncate
		endIndex := len(blocks)
		for j := len(blocks) - 1; j >= 0; j-- {
			m, ok := blocks[j].(map[string]interface{})
			if !ok {
				break
			}

			blockType, _ := m["type"].(string)
			if blockType != "thinking" {
				break
			}

			// Check signature
			if !hasValidThinkingSignature(m) {
				endIndex = j
			} else {
				break // Valid thinking block, stop scanning
			}
		}

		if endIndex < len(blocks) {
			msg.Content = blocks[:endIndex]
		}
	}
}

// hasValidSignatureForFunctionCalls checks if we have any valid signature available for function calls
// [FIX #295] This prevents Gemini 3 Pro from rejecting requests due to missing thought_signature
func hasValidSignatureForFunctionCalls(messages []ClaudeMessage, globalSig string) bool {
	// 1. Check global store
	if len(globalSig) >= MinSignatureLength {
		return true
	}

	// 2. Check if any message has a thinking block with valid signature
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != "assistant" {
			continue
		}

		blocks, ok := msg.Content.([]interface{})
		if !ok {
			continue
		}

		for _, block := range blocks {
			m, ok := block.(map[string]interface{})
			if !ok {
				continue
			}

			blockType, _ := m["type"].(string)
			if blockType == "thinking" {
				if sig, ok := m["signature"].(string); ok && len(sig) >= MinSignatureLength {
					return true
				}
			}
		}
	}
	return false
}

// hasThinkingHistory checks if there are any thinking blocks in message history
func hasThinkingHistory(messages []ClaudeMessage) bool {
	for _, msg := range messages {
		if msg.Role != "assistant" {
			continue
		}

		blocks, ok := msg.Content.([]interface{})
		if !ok {
			continue
		}

		for _, block := range blocks {
			if m, ok := block.(map[string]interface{}); ok {
				if blockType, _ := m["type"].(string); blockType == "thinking" {
					return true
				}
			}
		}
	}
	return false
}

// hasFunctionCalls checks if there are any tool_use blocks in messages
func hasFunctionCalls(messages []ClaudeMessage) bool {
	for _, msg := range messages {
		blocks, ok := msg.Content.([]interface{})
		if !ok {
			continue
		}

		for _, block := range blocks {
			if m, ok := block.(map[string]interface{}); ok {
				if blockType, _ := m["type"].(string); blockType == "tool_use" {
					return true
				}
			}
		}
	}
	return false
}

// shouldDisableThinkingDueToHistory checks if thinking should be disabled
// due to incompatible tool-use history (like Antigravity-Manager)
func shouldDisableThinkingDueToHistory(messages []ClaudeMessage) bool {
	// Reverse iterate to find last assistant message
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != "assistant" {
			continue
		}

		// Check if content is array
		blocks, ok := msg.Content.([]interface{})
		if !ok {
			return false
		}

		hasToolUse := false
		hasThinking := false

		for _, block := range blocks {
			if m, ok := block.(map[string]interface{}); ok {
				blockType, _ := m["type"].(string)
				if blockType == "tool_use" {
					hasToolUse = true
				}
				if blockType == "thinking" {
					hasThinking = true
				}
			}
		}

		// If has tool_use but no thinking -> incompatible
		if hasToolUse && !hasThinking {
			return true
		}

		// Only check the last assistant message
		return false
	}
	return false
}

// shouldEnableThinkingByDefault checks if thinking mode should be enabled by default
// Claude Code v2.0.67+ enables thinking by default for Opus 4.5 models
func shouldEnableThinkingByDefault(model string) bool {
	modelLower := strings.ToLower(model)
	// Enable thinking by default for Opus 4.5 variants
	if strings.Contains(modelLower, "opus-4-6") || strings.Contains(modelLower, "opus-4.6") || strings.Contains(modelLower, "opus-4-5") || strings.Contains(modelLower, "opus-4.5") {
		return true
	}
	// Also enable for explicit thinking model variants
	if strings.Contains(modelLower, "-thinking") {
		return true
	}
	return false
}

// targetModelSupportsThinking checks if the target model supports thinking mode
func targetModelSupportsThinking(mappedModel string) bool {
	// Only models with "-thinking" suffix or Claude models support thinking
	return strings.Contains(mappedModel, "-thinking") || strings.HasPrefix(mappedModel, "claude-")
}

// hasWebSearchTool checks if any tool is a web search tool (like Antigravity-Manager)
func hasWebSearchTool(tools []ClaudeTool) bool {
	for _, tool := range tools {
		if tool.IsWebSearch() {
			return true
		}
	}
	return false
}

func mergeAdjacentRoles(contents []GeminiContent) []GeminiContent {
	if len(contents) == 0 {
		return contents
	}

	var merged []GeminiContent
	current := contents[0]

	for i := 1; i < len(contents); i++ {
		next := contents[i]
		if current.Role == next.Role {
			// Merge parts
			current.Parts = append(current.Parts, next.Parts...)
		} else {
			merged = append(merged, current)
			current = next
		}
	}
	merged = append(merged, current)

	return merged
}
