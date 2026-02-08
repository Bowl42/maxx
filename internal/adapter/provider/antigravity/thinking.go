package antigravity

import (
	"encoding/json"
	"strings"
)

// shouldEnableThinkingByDefault checks if thinking should be enabled by default for a model
// Reference: Antigravity-Manager's should_enable_thinking_by_default (line 380-398)
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

// HasThinkingEnabledWithModel checks if thinking is enabled, considering model defaults
// Claude Code v2.0.67+ enables thinking by default for Opus 4.5 models
func HasThinkingEnabledWithModel(requestBody []byte, model string) bool {
	// First check explicit thinking config
	if HasThinkingEnabled(requestBody) {
		return true
	}

	// Check model defaults using shouldEnableThinkingByDefault
	return shouldEnableThinkingByDefault(model)
}

// TargetModelSupportsThinking checks if the target model supports thinking mode
// (like Antigravity-Manager's target_model_supports_thinking)
func TargetModelSupportsThinking(mappedModel string) bool {
	modelLower := strings.ToLower(mappedModel)

	// Models with "-thinking" suffix support thinking
	if strings.Contains(modelLower, "-thinking") {
		return true
	}

	// Claude models support thinking
	if strings.HasPrefix(modelLower, "claude-") {
		return true
	}

	// Regular Gemini 2.5/3 models do NOT support thinking without -thinking suffix
	// Reference: Antigravity-Manager is more conservative (line 183-184)
	// Only models with explicit "-thinking" suffix are considered compatible
	return false
}

// ShouldDisableThinkingDueToHistory checks if thinking should be disabled
// due to incompatible tool-use history (like Antigravity-Manager)
// Scenario: last Assistant message has ToolUse but no Thinking block
func ShouldDisableThinkingDueToHistory(contents []interface{}) bool {
	// Reverse iterate to find last model/assistant message
	for i := len(contents) - 1; i >= 0; i-- {
		content, ok := contents[i].(map[string]interface{})
		if !ok {
			continue
		}

		role, _ := content["role"].(string)
		if role != "model" && role != "assistant" {
			continue
		}

		// Found last assistant message
		parts, ok := content["parts"].([]interface{})
		if !ok {
			return false
		}

		hasToolUse := false
		hasThinking := false

		for _, part := range parts {
			partMap, ok := part.(map[string]interface{})
			if !ok {
				continue
			}

			// Check for functionCall (tool_use in Gemini format)
			if _, hasFc := partMap["functionCall"]; hasFc {
				hasToolUse = true
			}

			// Check for thought: true (thinking block in Gemini format)
			if thought, ok := partMap["thought"].(bool); ok && thought {
				hasThinking = true
			}
		}

		// If has tool_use but no thinking -> incompatible, must disable thinking
		if hasToolUse && !hasThinking {
			return true
		}

		// Only check the last assistant message
		return false
	}

	return false
}

// ValidateThinkingBlockPosition validates that thinking blocks are first in parts
// and handles invalid positions by downgrading to text
// (like Antigravity-Manager's thinking block position validation)
func ValidateThinkingBlockPosition(parts []interface{}) []interface{} {
	var result []interface{}
	foundNonThinking := false

	for _, part := range parts {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			result = append(result, part)
			foundNonThinking = true
			continue
		}

		thought, isThought := partMap["thought"].(bool)
		if isThought && thought {
			// This is a thinking block
			if foundNonThinking {
				// Thinking block at non-zero index - downgrade to text
				text, _ := partMap["text"].(string)
				if text != "" {
					result = append(result, map[string]interface{}{"text": text})
				}
			} else {
				// Thinking block at correct position
				result = append(result, part)
			}
		} else {
			// Non-thinking block
			foundNonThinking = true
			result = append(result, part)
		}
	}

	return result
}

// HandleEmptyThinkingBlocks fills empty thinking blocks with "..."
// (like Antigravity-Manager's empty thinking block handling)
func HandleEmptyThinkingBlocks(parts []interface{}) []interface{} {
	var result []interface{}

	for _, part := range parts {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			result = append(result, part)
			continue
		}

		thought, isThought := partMap["thought"].(bool)
		if isThought && thought {
			text, _ := partMap["text"].(string)
			if text == "" {
				// Empty thinking block - downgrade to text with "..."
				result = append(result, map[string]interface{}{"text": "..."})
				continue
			}
		}

		result = append(result, part)
	}

	return result
}

// ConvertRedactedThinking converts RedactedThinking blocks to text format
// (like Antigravity-Manager's RedactedThinking handling)
func ConvertRedactedThinking(parts []interface{}) []interface{} {
	var result []interface{}

	for _, part := range parts {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			result = append(result, part)
			continue
		}

		// Check for redacted_thinking type
		blockType, _ := partMap["type"].(string)
		if blockType == "redacted_thinking" {
			data, _ := partMap["data"].(string)
			result = append(result, map[string]interface{}{
				"text": "[Redacted Thinking: " + data + "]",
			})
			continue
		}

		result = append(result, part)
	}

	return result
}

// ProcessThinkingBlocks applies all thinking block validations and transformations
// (combines position validation, empty handling, and redacted thinking conversion)
func ProcessThinkingBlocks(contents []interface{}, hasThinking bool) {
	for _, content := range contents {
		contentMap, ok := content.(map[string]interface{})
		if !ok {
			continue
		}

		parts, ok := contentMap["parts"].([]interface{})
		if !ok {
			continue
		}

		// 1. Convert redacted thinking
		parts = ConvertRedactedThinking(parts)

		if hasThinking {
			// 2. Validate thinking block position
			parts = ValidateThinkingBlockPosition(parts)

			// 3. Handle empty thinking blocks
			parts = HandleEmptyThinkingBlocks(parts)
		}

		contentMap["parts"] = parts
	}
}

// ExtractThinkingConfig extracts thinking configuration from Claude request
func ExtractThinkingConfig(requestBody []byte) (enabled bool, budgetTokens int) {
	var req map[string]interface{}
	if err := json.Unmarshal(requestBody, &req); err != nil {
		return false, 0
	}

	thinking, ok := req["thinking"].(map[string]interface{})
	if !ok {
		return false, 0
	}

	thinkingType, _ := thinking["type"].(string)
	if thinkingType != "enabled" {
		return false, 0
	}

	budget, _ := thinking["budget_tokens"].(float64)
	return true, int(budget)
}
