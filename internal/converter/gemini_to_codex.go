package converter

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/awsl-project/maxx/internal/domain"
)

func init() {
	RegisterConverter(domain.ClientTypeGemini, domain.ClientTypeCodex, &geminiToCodexRequest{}, &geminiToCodexResponse{})
}

type geminiToCodexRequest struct{}
type geminiToCodexResponse struct{}

func (c *geminiToCodexRequest) Transform(body []byte, model string, stream bool) ([]byte, error) {
	var req GeminiRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	codexReq := CodexRequest{
		Model:  model,
		Stream: stream,
	}

	// Convert generation config
	if req.GenerationConfig != nil {
		codexReq.MaxOutputTokens = req.GenerationConfig.MaxOutputTokens
		codexReq.Temperature = req.GenerationConfig.Temperature
		codexReq.TopP = req.GenerationConfig.TopP
	}

	// Convert system instruction to instructions
	if req.SystemInstruction != nil {
		var systemText string
		for _, part := range req.SystemInstruction.Parts {
			if part.Text != "" {
				systemText += part.Text
			}
		}
		codexReq.Instructions = systemText
	}

	// Convert contents to input
	var inputItems []map[string]interface{}
	for _, content := range req.Contents {
		role := mapGeminiRoleToCodex(content.Role)
		var contentParts []map[string]interface{}

		for _, part := range content.Parts {
			if part.Text != "" {
				partType := "input_text"
				if role == "assistant" {
					partType = "output_text"
				}
				contentParts = append(contentParts, map[string]interface{}{
					"type": partType,
					"text": part.Text,
				})
			}
			if part.FunctionCall != nil {
				argsJSON, _ := json.Marshal(part.FunctionCall.Args)
				// Extract call_id from name if present
				name := part.FunctionCall.Name
				callID := "call_" + time.Now().Format("20060102150405")
				if idx := strings.LastIndex(name, "_"); idx > 0 {
					possibleID := name[idx+1:]
					if strings.HasPrefix(possibleID, "call_") {
						callID = possibleID
						name = name[:idx]
					}
				}
				inputItems = append(inputItems, map[string]interface{}{
					"type":      "function_call",
					"name":      name,
					"call_id":   callID,
					"arguments": string(argsJSON),
				})
				continue
			}
			if part.FunctionResponse != nil {
				// Extract call_id from name
				name := part.FunctionResponse.Name
				callID := "call_" + time.Now().Format("20060102150405")
				if idx := strings.LastIndex(name, "_"); idx > 0 {
					possibleID := name[idx+1:]
					if strings.HasPrefix(possibleID, "call_") {
						callID = possibleID
					}
				}
				respJSON, _ := json.Marshal(part.FunctionResponse.Response)
				inputItems = append(inputItems, map[string]interface{}{
					"type":    "function_call_output",
					"call_id": callID,
					"output":  string(respJSON),
				})
				continue
			}
		}

		if len(contentParts) > 0 {
			inputItems = append(inputItems, map[string]interface{}{
				"type":    "message",
				"role":    role,
				"content": contentParts,
			})
		}
	}

	if len(inputItems) == 1 {
		// Check if single text message from user
		item := inputItems[0]
		if item["type"] == "message" && item["role"] == "user" {
			if content, ok := item["content"].([]map[string]interface{}); ok {
				if len(content) == 1 && content[0]["type"] == "input_text" {
					codexReq.Input = content[0]["text"]
					goto skipInputItems
				}
			}
		}
	}
	codexReq.Input = inputItems
skipInputItems:

	// Convert tools
	for _, tool := range req.Tools {
		for _, funcDecl := range tool.FunctionDeclarations {
			codexReq.Tools = append(codexReq.Tools, CodexTool{
				Type:        "function",
				Name:        funcDecl.Name,
				Description: funcDecl.Description,
				Parameters:  funcDecl.Parameters,
			})
		}
	}

	return json.Marshal(codexReq)
}

func mapGeminiRoleToCodex(role string) string {
	switch role {
	case "user":
		return "user"
	case "model":
		return "assistant"
	default:
		return "user"
	}
}

func (c *geminiToCodexResponse) Transform(body []byte) ([]byte, error) {
	var resp CodexResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	geminiResp := GeminiResponse{
		UsageMetadata: &GeminiUsageMetadata{
			PromptTokenCount:     resp.Usage.InputTokens,
			CandidatesTokenCount: resp.Usage.OutputTokens,
			TotalTokenCount:      resp.Usage.TotalTokens,
		},
	}

	// Convert output to candidates
	var parts []GeminiPart
	for _, out := range resp.Output {
		switch out.Type {
		case "message":
			switch content := out.Content.(type) {
			case string:
				parts = append(parts, GeminiPart{Text: content})
			case []interface{}:
				for _, c := range content {
					if cm, ok := c.(map[string]interface{}); ok {
						if text, ok := cm["text"].(string); ok {
							parts = append(parts, GeminiPart{Text: text})
						}
					}
				}
			}
		case "function_call":
			var args map[string]interface{}
			json.Unmarshal([]byte(out.Arguments), &args)
			// Embed call_id in name for round-trip
			name := out.Name
			if out.CallID != "" {
				name = out.Name + "_" + out.CallID
			}
			parts = append(parts, GeminiPart{
				FunctionCall: &GeminiFunctionCall{
					Name: name,
					Args: args,
				},
			})
		}
	}

	finishReason := "STOP"
	if resp.Status == "incomplete" {
		finishReason = "MAX_TOKENS"
	}
	// Check if there are function calls
	for _, part := range parts {
		if part.FunctionCall != nil {
			finishReason = "STOP"
			break
		}
	}

	geminiResp.Candidates = []GeminiCandidate{{
		Content: GeminiContent{
			Role:  "model",
			Parts: parts,
		},
		FinishReason: finishReason,
		Index:        0,
	}}

	return json.Marshal(geminiResp)
}

func (c *geminiToCodexResponse) TransformChunk(chunk []byte, state *TransformState) ([]byte, error) {
	events, remaining := ParseSSE(state.Buffer + string(chunk))
	state.Buffer = remaining

	var output []byte
	for _, event := range events {
		if event.Event == "done" {
			continue
		}

		var codexEvent CodexStreamEvent
		if err := json.Unmarshal(event.Data, &codexEvent); err != nil {
			continue
		}

		switch codexEvent.Type {
		case "response.created":
			if codexEvent.Response != nil {
				state.MessageID = codexEvent.Response.ID
			}

		case "response.output_text.delta":
			if codexEvent.Delta != nil && codexEvent.Delta.Text != "" {
				geminiChunk := GeminiStreamChunk{
					Candidates: []GeminiCandidate{{
						Content: GeminiContent{
							Role:  "model",
							Parts: []GeminiPart{{Text: codexEvent.Delta.Text}},
						},
						Index: 0,
					}},
				}
				output = append(output, FormatSSE("", geminiChunk)...)
			}

		case "response.output_item.added":
			if codexEvent.Item != nil && codexEvent.Item.Type == "function_call" {
				var args map[string]interface{}
				json.Unmarshal([]byte(codexEvent.Item.Arguments), &args)
				name := codexEvent.Item.Name
				if codexEvent.Item.CallID != "" {
					name = codexEvent.Item.Name + "_" + codexEvent.Item.CallID
				}
				geminiChunk := GeminiStreamChunk{
					Candidates: []GeminiCandidate{{
						Content: GeminiContent{
							Role: "model",
							Parts: []GeminiPart{{
								FunctionCall: &GeminiFunctionCall{
									Name: name,
									Args: args,
								},
							}},
						},
						Index: 0,
					}},
				}
				output = append(output, FormatSSE("", geminiChunk)...)
			}

		case "response.completed":
			if codexEvent.Response != nil {
				finishReason := "STOP"
				geminiChunk := GeminiStreamChunk{
					Candidates: []GeminiCandidate{{
						Content:      GeminiContent{Role: "model", Parts: []GeminiPart{}},
						FinishReason: finishReason,
						Index:        0,
					}},
					UsageMetadata: &GeminiUsageMetadata{
						PromptTokenCount:     codexEvent.Response.Usage.InputTokens,
						CandidatesTokenCount: codexEvent.Response.Usage.OutputTokens,
						TotalTokenCount:      codexEvent.Response.Usage.TotalTokens,
					},
				}
				output = append(output, FormatSSE("", geminiChunk)...)
			}
		}
	}

	return output, nil
}
