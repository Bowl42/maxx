package converter

import (
	"encoding/json"

	"github.com/awsl-project/maxx/internal/domain"
)

func init() {
	RegisterConverter(domain.ClientTypeOpenAI, domain.ClientTypeGemini, &openaiToGeminiRequest{}, &openaiToGeminiResponse{})
}

type openaiToGeminiRequest struct{}
type openaiToGeminiResponse struct{}

func (c *openaiToGeminiRequest) Transform(body []byte, model string, stream bool) ([]byte, error) {
	var req OpenAIRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	geminiReq := GeminiRequest{
		GenerationConfig: &GeminiGenerationConfig{
			MaxOutputTokens: req.MaxTokens,
			Temperature:     req.Temperature,
			TopP:            req.TopP,
		},
	}

	if req.MaxCompletionTokens > 0 && req.MaxTokens == 0 {
		geminiReq.GenerationConfig.MaxOutputTokens = req.MaxCompletionTokens
	}

	// Convert stop sequences
	switch stop := req.Stop.(type) {
	case string:
		geminiReq.GenerationConfig.StopSequences = []string{stop}
	case []interface{}:
		for _, s := range stop {
			if str, ok := s.(string); ok {
				geminiReq.GenerationConfig.StopSequences = append(geminiReq.GenerationConfig.StopSequences, str)
			}
		}
	}

	// Convert messages
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			var systemText string
			if content, ok := msg.Content.(string); ok {
				systemText = content
			}
			if systemText != "" {
				// [FIX] Set role to "user" for systemInstruction (like CLIProxyAPI)
				geminiReq.SystemInstruction = &GeminiContent{
					Role:  "user",
					Parts: []GeminiPart{{Text: systemText}},
				}
			}
			continue
		}

		geminiContent := GeminiContent{}
		switch msg.Role {
		case "user":
			geminiContent.Role = "user"
		case "assistant":
			geminiContent.Role = "model"
		case "tool":
			geminiContent.Role = "user"
			contentStr, _ := msg.Content.(string)
			geminiContent.Parts = []GeminiPart{{
				FunctionResponse: &GeminiFunctionResponse{
					Name:     msg.ToolCallID,
					Response: map[string]string{"result": contentStr},
				},
			}}
			geminiReq.Contents = append(geminiReq.Contents, geminiContent)
			continue
		}

		// Regular message content
		switch content := msg.Content.(type) {
		case string:
			geminiContent.Parts = []GeminiPart{{Text: content}}
		case []interface{}:
			for _, part := range content {
				if m, ok := part.(map[string]interface{}); ok {
					if m["type"] == "text" {
						if text, ok := m["text"].(string); ok {
							geminiContent.Parts = append(geminiContent.Parts, GeminiPart{Text: text})
						}
					}
				}
			}
		}

		// Handle tool calls
		for _, tc := range msg.ToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
			geminiContent.Parts = append(geminiContent.Parts, GeminiPart{
				FunctionCall: &GeminiFunctionCall{
					Name: tc.Function.Name,
					Args: args,
				},
			})
		}

		geminiReq.Contents = append(geminiReq.Contents, geminiContent)
	}

	// Convert tools
	if len(req.Tools) > 0 {
		var funcDecls []GeminiFunctionDecl
		for _, tool := range req.Tools {
			funcDecls = append(funcDecls, GeminiFunctionDecl{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			})
		}
		geminiReq.Tools = []GeminiTool{{FunctionDeclarations: funcDecls}}
	}

	return json.Marshal(geminiReq)
}

func (c *openaiToGeminiResponse) Transform(body []byte) ([]byte, error) {
	var resp OpenAIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	geminiResp := GeminiResponse{
		UsageMetadata: &GeminiUsageMetadata{
			PromptTokenCount:     resp.Usage.PromptTokens,
			CandidatesTokenCount: resp.Usage.CompletionTokens,
			TotalTokenCount:      resp.Usage.TotalTokens,
		},
	}

	candidate := GeminiCandidate{
		Content: GeminiContent{Role: "model"},
		Index:   0,
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if choice.Message != nil {
			if content, ok := choice.Message.Content.(string); ok && content != "" {
				candidate.Content.Parts = append(candidate.Content.Parts, GeminiPart{Text: content})
			}
			for _, tc := range choice.Message.ToolCalls {
				var args map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				candidate.Content.Parts = append(candidate.Content.Parts, GeminiPart{
					FunctionCall: &GeminiFunctionCall{
						Name: tc.Function.Name,
						Args: args,
					},
				})
			}

			switch choice.FinishReason {
			case "stop":
				candidate.FinishReason = "STOP"
			case "length":
				candidate.FinishReason = "MAX_TOKENS"
			case "tool_calls":
				candidate.FinishReason = "STOP"
			}
		}
	}

	geminiResp.Candidates = []GeminiCandidate{candidate}
	return json.Marshal(geminiResp)
}

func (c *openaiToGeminiResponse) TransformChunk(chunk []byte, state *TransformState) ([]byte, error) {
	events, remaining := ParseSSE(state.Buffer + string(chunk))
	state.Buffer = remaining

	var output []byte
	for _, event := range events {
		if event.Event == "done" {
			continue
		}

		var openaiChunk OpenAIStreamChunk
		if err := json.Unmarshal(event.Data, &openaiChunk); err != nil {
			continue
		}

		if len(openaiChunk.Choices) > 0 {
			choice := openaiChunk.Choices[0]
			if choice.Delta != nil {
				if content, ok := choice.Delta.Content.(string); ok && content != "" {
					geminiChunk := GeminiStreamChunk{
						Candidates: []GeminiCandidate{{
							Content: GeminiContent{
								Role:  "model",
								Parts: []GeminiPart{{Text: content}},
							},
							Index: 0,
						}},
					}
					output = append(output, FormatSSE("", geminiChunk)...)
				}
			}

			if choice.FinishReason != "" {
				finishReason := "STOP"
				if choice.FinishReason == "length" {
					finishReason = "MAX_TOKENS"
				}
				geminiChunk := GeminiStreamChunk{
					Candidates: []GeminiCandidate{{
						FinishReason: finishReason,
						Index:        0,
					}},
				}
				output = append(output, FormatSSE("", geminiChunk)...)
			}
		}
	}

	return output, nil
}
