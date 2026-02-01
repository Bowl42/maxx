package converter

import (
	"encoding/json"
	"strings"

	"github.com/awsl-project/maxx/internal/domain"
)

func init() {
	RegisterConverter(domain.ClientTypeOpenAI, domain.ClientTypeClaude, &openaiToClaudeRequest{}, &openaiToClaudeResponse{})
}

type openaiToClaudeRequest struct{}

func (c *openaiToClaudeRequest) Transform(body []byte, model string, stream bool) ([]byte, error) {
	var req OpenAIRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	claudeReq := ClaudeRequest{
		Model:       model,
		Stream:      stream,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	if req.MaxCompletionTokens > 0 && req.MaxTokens == 0 {
		claudeReq.MaxTokens = req.MaxCompletionTokens
	}

	// Convert messages
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			// Extract system message
			switch content := msg.Content.(type) {
			case string:
				claudeReq.System = content
			case []interface{}:
				var systemText string
				for _, part := range content {
					if m, ok := part.(map[string]interface{}); ok {
						if text, ok := m["text"].(string); ok {
							systemText += text
						}
					}
				}
				claudeReq.System = systemText
			}
			continue
		}

		claudeMsg := ClaudeMessage{Role: msg.Role}

		// Handle tool messages
		if msg.Role == "tool" {
			claudeMsg.Role = "user"
			contentStr, _ := msg.Content.(string)
			claudeMsg.Content = []ClaudeContentBlock{{
				Type:      "tool_result",
				ToolUseID: msg.ToolCallID,
				Content:   contentStr,
			}}
			claudeReq.Messages = append(claudeReq.Messages, claudeMsg)
			continue
		}

		var blocks []ClaudeContentBlock

		// Convert reasoning_content to thinking blocks (assistant only)
		if msg.Role == "assistant" {
			if thinkingText := collectReasoningText(msg.ReasoningContent); strings.TrimSpace(thinkingText) != "" {
				blocks = append(blocks, ClaudeContentBlock{Type: "thinking", Thinking: thinkingText})
			}
		}

		// Convert content
		switch content := msg.Content.(type) {
		case string:
			if content != "" {
				blocks = append(blocks, ClaudeContentBlock{Type: "text", Text: content})
			}
		case []interface{}:
			for _, part := range content {
				if m, ok := part.(map[string]interface{}); ok {
					partType, _ := m["type"].(string)
					switch partType {
					case "text":
						text, _ := m["text"].(string)
						if text != "" {
							blocks = append(blocks, ClaudeContentBlock{Type: "text", Text: text})
						}
					}
				}
			}
		}

		// Handle tool calls
		for _, tc := range msg.ToolCalls {
			var input interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &input)
			blocks = append(blocks, ClaudeContentBlock{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: input,
			})
		}

		if len(blocks) == 1 && blocks[0].Type == "text" {
			claudeMsg.Content = blocks[0].Text
		} else {
			claudeMsg.Content = blocks
		}

		claudeReq.Messages = append(claudeReq.Messages, claudeMsg)
	}

	// Convert tools
	for _, tool := range req.Tools {
		claudeReq.Tools = append(claudeReq.Tools, ClaudeTool{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			InputSchema: tool.Function.Parameters,
		})
	}

	// Convert stop
	switch stop := req.Stop.(type) {
	case string:
		claudeReq.StopSequences = []string{stop}
	case []interface{}:
		for _, s := range stop {
			if str, ok := s.(string); ok {
				claudeReq.StopSequences = append(claudeReq.StopSequences, str)
			}
		}
	}

	return json.Marshal(claudeReq)
}
