package converter

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/awsl-project/maxx/internal/domain"
)

func init() {
	RegisterConverter(domain.ClientTypeClaude, domain.ClientTypeCodex, &claudeToCodexRequest{}, &claudeToCodexResponse{})
}

type claudeToCodexRequest struct{}
type claudeToCodexResponse struct{}

func (c *claudeToCodexRequest) Transform(body []byte, model string, stream bool) ([]byte, error) {
	userAgent := ExtractCodexUserAgent(body)
	var req ClaudeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	codexReq := CodexRequest{
		Model:           model,
		Stream:          stream,
		MaxOutputTokens: req.MaxTokens,
		Temperature:     req.Temperature,
		TopP:            req.TopP,
	}

	shortMap := map[string]string{}
	if len(req.Tools) > 0 {
		var names []string
		for _, tool := range req.Tools {
			if tool.Type != "" {
				continue // server tools should keep their type
			}
			if tool.Name != "" {
				names = append(names, tool.Name)
			}
		}
		if len(names) > 0 {
			shortMap = buildShortNameMap(names)
		}
	}

	// Convert messages to input
	var input []CodexInputItem
	if req.System != nil {
		switch s := req.System.(type) {
		case string:
			if s != "" {
				input = append(input, CodexInputItem{Type: "message", Role: "developer", Content: s})
			}
		case []interface{}:
			var systemText string
			for _, block := range s {
				if m, ok := block.(map[string]interface{}); ok {
					if text, ok := m["text"].(string); ok {
						systemText += text
					}
				}
			}
			if systemText != "" {
				input = append(input, CodexInputItem{Type: "message", Role: "developer", Content: systemText})
			}
		}
	}
	for _, msg := range req.Messages {
		item := CodexInputItem{Role: msg.Role}
		switch content := msg.Content.(type) {
		case string:
			item.Type = "message"
			item.Content = content
		case []interface{}:
			for _, block := range content {
				if m, ok := block.(map[string]interface{}); ok {
					blockType, _ := m["type"].(string)
					switch blockType {
					case "text":
						item.Type = "message"
						item.Content = m["text"]
					case "tool_use":
						// Convert tool use to function_call output
						name, _ := m["name"].(string)
						if short, ok := shortMap[name]; ok {
							name = short
						} else {
							name = shortenNameIfNeeded(name)
						}
						id, _ := m["id"].(string)
						inputData := m["input"]
						argJSON, _ := json.Marshal(inputData)
						input = append(input, CodexInputItem{
							Type:      "function_call",
							ID:        id,
							CallID:    id,
							Name:      name,
							Role:      "assistant",
							Arguments: string(argJSON),
						})
						continue
					case "tool_result":
						toolUseID, _ := m["tool_use_id"].(string)
						resultContent, _ := m["content"].(string)
						input = append(input, CodexInputItem{
							Type:   "function_call_output",
							CallID: toolUseID,
							Output: resultContent,
						})
						continue
					}
				}
			}
		}
		if item.Type != "" {
			input = append(input, item)
		}
	}
	codexReq.Input = input

	// Convert tools
	for _, tool := range req.Tools {
		if tool.Type != "" {
			codexReq.Tools = append(codexReq.Tools, CodexTool{
				Type: tool.Type,
			})
			continue
		}
		name := tool.Name
		if short, ok := shortMap[name]; ok {
			name = short
		} else {
			name = shortenNameIfNeeded(name)
		}
		codexReq.Tools = append(codexReq.Tools, CodexTool{
			Type:        "function",
			Name:        name,
			Description: tool.Description,
			Parameters:  tool.InputSchema,
		})
	}

	if req.OutputConfig != nil {
		effort := strings.ToLower(strings.TrimSpace(req.OutputConfig.Effort))
		codexReq.Reasoning = &CodexReasoning{Effort: effort}
	}
	if instructions := CodexInstructionsForModel(model, userAgent); instructions != "" {
		codexReq.Instructions = instructions
	}
	if codexReq.Reasoning == nil {
		codexReq.Reasoning = &CodexReasoning{Effort: "medium", Summary: "auto"}
	} else if codexReq.Reasoning.Summary == "" {
		codexReq.Reasoning.Summary = "auto"
	}
	if codexReq.Reasoning.Effort == "" {
		codexReq.Reasoning.Effort = "medium"
	}

	return json.Marshal(codexReq)
}

func (c *claudeToCodexResponse) Transform(body []byte) ([]byte, error) {
	var resp ClaudeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	codexResp := CodexResponse{
		ID:        resp.ID,
		Object:    "response",
		CreatedAt: time.Now().Unix(),
		Model:     resp.Model,
		Status:    "completed",
		Usage: CodexUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	// Convert content to output
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			codexResp.Output = append(codexResp.Output, CodexOutput{
				Type:    "message",
				Role:    "assistant",
				Content: block.Text,
			})
		case "tool_use":
			argJSON, _ := json.Marshal(block.Input)
			codexResp.Output = append(codexResp.Output, CodexOutput{
				Type:      "function_call",
				ID:        block.ID,
				CallID:    block.ID,
				Name:      block.Name,
				Arguments: string(argJSON),
				Status:    "completed",
			})
		}
	}

	return json.Marshal(codexResp)
}

func (c *claudeToCodexResponse) TransformChunk(chunk []byte, state *TransformState) ([]byte, error) {
	events, remaining := ParseSSE(state.Buffer + string(chunk))
	state.Buffer = remaining

	var output []byte
	for _, event := range events {
		if event.Event == "done" {
			output = append(output, FormatSSE("", map[string]string{"type": "response.done"})...)
			continue
		}

		var claudeEvent ClaudeStreamEvent
		if err := json.Unmarshal(event.Data, &claudeEvent); err != nil {
			continue
		}

		switch claudeEvent.Type {
		case "message_start":
			if claudeEvent.Message != nil {
				state.MessageID = claudeEvent.Message.ID
			}
			codexEvent := map[string]interface{}{
				"type": "response.created",
				"response": map[string]interface{}{
					"id":     state.MessageID,
					"status": "in_progress",
				},
			}
			output = append(output, FormatSSE("", codexEvent)...)

		case "content_block_delta":
			if claudeEvent.Delta != nil && claudeEvent.Delta.Type == "text_delta" {
				codexEvent := map[string]interface{}{
					"type": "response.output_item.delta",
					"delta": map[string]interface{}{
						"type": "text",
						"text": claudeEvent.Delta.Text,
					},
				}
				output = append(output, FormatSSE("", codexEvent)...)
			}

		case "message_stop":
			codexEvent := map[string]interface{}{
				"type": "response.done",
				"response": map[string]interface{}{
					"id":     state.MessageID,
					"status": "completed",
				},
			}
			output = append(output, FormatSSE("", codexEvent)...)
		}
	}

	return output, nil
}
