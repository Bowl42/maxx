package converter

import (
	"encoding/json"
	"strings"
)

func extractClaudeThinkingText(block map[string]interface{}) string {
	if thinking, ok := block["thinking"].(string); ok {
		return thinking
	}
	if text, ok := block["text"].(string); ok {
		return text
	}
	return ""
}

func convertClaudeToolResultContentToString(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var sb strings.Builder
		for _, part := range v {
			if m, ok := part.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					sb.WriteString(text)
				}
			}
		}
		return sb.String()
	default:
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
	}
	return ""
}

func applyClaudeThinkingToOpenAI(openaiReq *OpenAIRequest, claudeReq *ClaudeRequest) {
	if openaiReq == nil || claudeReq == nil {
		return
	}
	if claudeReq.OutputConfig != nil && claudeReq.OutputConfig.Effort != "" {
		openaiReq.ReasoningEffort = claudeReq.OutputConfig.Effort
		return
	}
	if claudeReq.Thinking == nil {
		return
	}
	thinkingType, _ := claudeReq.Thinking["type"].(string)
	switch thinkingType {
	case "enabled":
		if budgetAny, ok := claudeReq.Thinking["budget_tokens"]; ok {
			if budget, ok := asInt(budgetAny); ok {
				if effort := mapBudgetToEffort(budget); effort != "" {
					openaiReq.ReasoningEffort = effort
				}
			}
		} else {
			openaiReq.ReasoningEffort = "auto"
		}
	case "disabled":
		openaiReq.ReasoningEffort = "none"
	}
}

func asInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

func mapBudgetToEffort(budget int) string {
	switch {
	case budget < 0:
		if budget == -1 {
			return "auto"
		}
		return ""
	case budget == 0:
		return "none"
	case budget <= 1024:
		return "low"
	case budget <= 8192:
		return "medium"
	default:
		return "high"
	}
}

// Add Index field to OpenAIToolCall for streaming
type OpenAIToolCallWithIndex struct {
	Index    int                `json:"index"`
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function OpenAIFunctionCall `json:"function,omitempty"`
}
