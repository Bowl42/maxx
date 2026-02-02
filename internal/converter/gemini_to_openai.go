package converter

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/awsl-project/maxx/internal/domain"
)

func init() {
	RegisterConverter(domain.ClientTypeGemini, domain.ClientTypeOpenAI, &geminiToOpenAIRequest{}, &geminiToOpenAIResponse{})
}

type geminiToOpenAIRequest struct{}
type geminiToOpenAIResponse struct{}

func (c *geminiToOpenAIRequest) Transform(body []byte, model string, stream bool) ([]byte, error) {
	var req GeminiRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	openaiReq := OpenAIRequest{
		Model:  model,
		Stream: stream,
	}

	if req.GenerationConfig != nil {
		openaiReq.MaxTokens = req.GenerationConfig.MaxOutputTokens
		openaiReq.Temperature = req.GenerationConfig.Temperature
		openaiReq.TopP = req.GenerationConfig.TopP
		if len(req.GenerationConfig.StopSequences) > 0 {
			openaiReq.Stop = req.GenerationConfig.StopSequences
		}
		if req.GenerationConfig.ThinkingConfig != nil {
			if req.GenerationConfig.ThinkingConfig.ThinkingLevel != "" {
				openaiReq.ReasoningEffort = strings.ToLower(req.GenerationConfig.ThinkingConfig.ThinkingLevel)
			} else {
				openaiReq.ReasoningEffort = mapBudgetToEffort(req.GenerationConfig.ThinkingConfig.ThinkingBudget)
			}
		}
	}

	// Convert systemInstruction
	if req.SystemInstruction != nil {
		var systemText string
		for _, part := range req.SystemInstruction.Parts {
			systemText += part.Text
		}
		if systemText != "" {
			openaiReq.Messages = append(openaiReq.Messages, OpenAIMessage{
				Role:    "system",
				Content: systemText,
			})
		}
	}

	// Convert contents to messages
	for _, content := range req.Contents {
		openaiMsg := OpenAIMessage{}
		switch content.Role {
		case "user":
			openaiMsg.Role = "user"
		case "model":
			openaiMsg.Role = "assistant"
		default:
			openaiMsg.Role = "user"
		}

		var textContent string
		var reasoningContent string
		var contentParts []OpenAIContentPart
		onlyText := true
		var toolCalls []OpenAIToolCall

		for _, part := range content.Parts {
			if part.Thought && part.Text != "" {
				reasoningContent += part.Text
			}
			if part.Text != "" {
				textContent += part.Text
				contentParts = append(contentParts, OpenAIContentPart{Type: "text", Text: part.Text})
			}
			if part.InlineData != nil && part.InlineData.Data != "" {
				onlyText = false
				contentParts = append(contentParts, OpenAIContentPart{
					Type:     "image_url",
					ImageURL: &OpenAIImageURL{URL: "data:" + part.InlineData.MimeType + ";base64," + part.InlineData.Data},
				})
			}
			if part.FunctionCall != nil {
				argsJSON, _ := json.Marshal(part.FunctionCall.Args)
				id := part.FunctionCall.ID
				if id == "" {
					id = "call_" + part.FunctionCall.Name
				}
				toolCalls = append(toolCalls, OpenAIToolCall{
					ID:   id,
					Type: "function",
					Function: OpenAIFunctionCall{
						Name:      part.FunctionCall.Name,
						Arguments: string(argsJSON),
					},
				})
			}
			if part.FunctionResponse != nil {
				respJSON, _ := json.Marshal(part.FunctionResponse.Response)
				toolName, callID := splitFunctionName(part.FunctionResponse.Name)
				if callID == "" {
					callID = part.FunctionResponse.ID
				}
				if callID == "" {
					callID = part.FunctionResponse.Name
				}
				openaiReq.Messages = append(openaiReq.Messages, OpenAIMessage{
					Role:       "tool",
					Content:    string(respJSON),
					ToolCallID: callID,
					Name:       toolName,
				})
				continue
			}
		}

		if onlyText && textContent != "" {
			openaiMsg.Content = textContent
		} else if len(contentParts) > 0 {
			openaiMsg.Content = contentParts
		}
		if reasoningContent != "" {
			openaiMsg.ReasoningContent = reasoningContent
		}
		if len(toolCalls) > 0 {
			openaiMsg.ToolCalls = toolCalls
		}

		if openaiMsg.Content != nil || len(openaiMsg.ToolCalls) > 0 {
			openaiReq.Messages = append(openaiReq.Messages, openaiMsg)
		}
	}

	// Convert tools
	for _, tool := range req.Tools {
		for _, decl := range tool.FunctionDeclarations {
			params := decl.Parameters
			if params == nil {
				params = decl.ParametersJsonSchema
			}
			openaiReq.Tools = append(openaiReq.Tools, OpenAITool{
				Type:     "function",
				Function: OpenAIFunction{Name: decl.Name, Description: decl.Description, Parameters: params},
			})
		}
	}

	return json.Marshal(openaiReq)
}

func (c *geminiToOpenAIResponse) Transform(body []byte) ([]byte, error) {
	var resp GeminiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	openaiResp := OpenAIResponse{
		ID:      "chatcmpl-gemini",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
	}

	if resp.UsageMetadata != nil {
		openaiResp.Usage = OpenAIUsage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	msg := OpenAIMessage{Role: "assistant"}
	var textContent string
	var reasoningContent string
	var toolCalls []OpenAIToolCall
	finishReason := "stop"

	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		for _, part := range candidate.Content.Parts {
			if part.Thought && part.Text != "" {
				reasoningContent += part.Text
				continue
			}
			if part.Text != "" {
				textContent += part.Text
			}
			if part.InlineData != nil && part.InlineData.Data != "" {
				if msg.Content == nil {
					msg.Content = []OpenAIContentPart{}
				}
				parts, _ := msg.Content.([]OpenAIContentPart)
				parts = append(parts, OpenAIContentPart{
					Type:     "image_url",
					ImageURL: &OpenAIImageURL{URL: "data:" + part.InlineData.MimeType + ";base64," + part.InlineData.Data},
				})
				msg.Content = parts
			}
			if part.FunctionCall != nil {
				argsJSON, _ := json.Marshal(part.FunctionCall.Args)
				id := part.FunctionCall.ID
				if id == "" {
					id = "call_" + part.FunctionCall.Name
				}
				toolCalls = append(toolCalls, OpenAIToolCall{
					ID:   id,
					Type: "function",
					Function: OpenAIFunctionCall{
						Name:      part.FunctionCall.Name,
						Arguments: string(argsJSON),
					},
				})
			}
		}

		switch candidate.FinishReason {
		case "STOP":
			if len(toolCalls) > 0 {
				finishReason = "tool_calls"
			} else {
				finishReason = "stop"
			}
		case "MAX_TOKENS":
			finishReason = "length"
		}
	}

	if textContent != "" {
		if msg.Content == nil {
			msg.Content = textContent
		} else if parts, ok := msg.Content.([]OpenAIContentPart); ok {
			parts = append(parts, OpenAIContentPart{Type: "text", Text: textContent})
			msg.Content = parts
		}
	}
	if reasoningContent != "" {
		msg.ReasoningContent = reasoningContent
	}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}

	openaiResp.Choices = []OpenAIChoice{{
		Index:        0,
		Message:      &msg,
		FinishReason: finishReason,
	}}

	return json.Marshal(openaiResp)
}

func (c *geminiToOpenAIResponse) TransformChunk(chunk []byte, state *TransformState) ([]byte, error) {
	events, remaining := ParseSSE(state.Buffer + string(chunk))
	state.Buffer = remaining

	var output []byte
	for _, event := range events {
		var geminiChunk GeminiStreamChunk
		if err := json.Unmarshal(event.Data, &geminiChunk); err != nil {
			continue
		}

		// First chunk
		if state.MessageID == "" {
			state.MessageID = "chatcmpl-gemini"
			if state.CreatedAt == 0 {
				state.CreatedAt = time.Now().Unix()
			}
			output = append(output, formatOpenAIStreamChunk(state, map[string]interface{}{
				"role":    "assistant",
				"content": "",
			}, nil)...)
		}

		if len(geminiChunk.Candidates) > 0 {
			candidate := geminiChunk.Candidates[0]
			for _, part := range candidate.Content.Parts {
				if part.Thought && part.Text != "" {
					output = append(output, formatOpenAIStreamChunk(state, map[string]interface{}{
						"reasoning_content": part.Text,
					}, nil)...)
					continue
				}
				if part.Text != "" {
					output = append(output, formatOpenAIStreamChunk(state, map[string]interface{}{
						"content": part.Text,
					}, nil)...)
				}
				if part.InlineData != nil && part.InlineData.Data != "" {
					output = append(output, formatOpenAIStreamChunk(state, map[string]interface{}{
						"content": []interface{}{
							map[string]interface{}{
								"type": "image_url",
								"image_url": map[string]interface{}{
									"url": "data:" + part.InlineData.MimeType + ";base64," + part.InlineData.Data,
								},
							},
						},
					}, nil)...)
				}
				if part.FunctionCall != nil {
					id := part.FunctionCall.ID
					if id == "" {
						id = "call_" + part.FunctionCall.Name
					}
					tool := map[string]interface{}{
						"index": state.CurrentIndex,
						"id":    id,
						"type":  "function",
						"function": map[string]interface{}{
							"name":      part.FunctionCall.Name,
							"arguments": string(mustMarshal(part.FunctionCall.Args)),
						},
					}
					state.CurrentIndex++
					output = append(output, formatOpenAIStreamChunk(state, map[string]interface{}{
						"tool_calls": []interface{}{tool},
					}, nil)...)
				}
			}

			if candidate.FinishReason != "" {
				finishReason := "stop"
				if candidate.FinishReason == "MAX_TOKENS" {
					finishReason = "length"
				}
				output = append(output, formatOpenAIStreamChunk(state, map[string]interface{}{}, &finishReason)...)
				output = append(output, FormatDone()...)
			}
		}
	}

	return output, nil
}

func splitFunctionName(name string) (string, string) {
	if idx := strings.LastIndex(name, "_call_"); idx > 0 {
		return name[:idx], name[idx+1:]
	}
	return name, ""
}
