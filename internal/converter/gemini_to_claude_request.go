package converter

import (
	"encoding/json"
	"fmt"

	"github.com/awsl-project/maxx/internal/domain"
)

func init() {
	RegisterConverter(domain.ClientTypeGemini, domain.ClientTypeClaude, &geminiToClaudeRequest{}, &geminiToClaudeResponse{})
}

type geminiToClaudeRequest struct{}

func (c *geminiToClaudeRequest) Transform(body []byte, model string, stream bool) ([]byte, error) {
	var req GeminiRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	claudeReq := ClaudeRequest{
		Model:  model,
		Stream: stream,
	}

	if req.GenerationConfig != nil {
		claudeReq.MaxTokens = req.GenerationConfig.MaxOutputTokens
		claudeReq.Temperature = req.GenerationConfig.Temperature
		claudeReq.TopP = req.GenerationConfig.TopP
		claudeReq.TopK = req.GenerationConfig.TopK
		claudeReq.StopSequences = req.GenerationConfig.StopSequences
	}

	// Convert systemInstruction
	if req.SystemInstruction != nil {
		var systemText string
		for _, part := range req.SystemInstruction.Parts {
			systemText += part.Text
		}
		if systemText != "" {
			claudeReq.System = systemText
		}
	}

	// Convert contents to messages
	toolCallCounter := 0
	for _, content := range req.Contents {
		claudeMsg := ClaudeMessage{}
		// Map role
		switch content.Role {
		case "user":
			claudeMsg.Role = "user"
		case "model":
			claudeMsg.Role = "assistant"
		default:
			claudeMsg.Role = "user"
		}

		var blocks []ClaudeContentBlock
		for _, part := range content.Parts {
			if part.Text != "" {
				blocks = append(blocks, ClaudeContentBlock{Type: "text", Text: part.Text})
			}
			if part.FunctionCall != nil {
				toolCallCounter++
				blocks = append(blocks, ClaudeContentBlock{
					Type:  "tool_use",
					ID:    fmt.Sprintf("call_%d", toolCallCounter),
					Name:  part.FunctionCall.Name,
					Input: part.FunctionCall.Args,
				})
			}
			if part.FunctionResponse != nil {
				respJSON, _ := json.Marshal(part.FunctionResponse.Response)
				blocks = append(blocks, ClaudeContentBlock{
					Type:      "tool_result",
					ToolUseID: part.FunctionResponse.Name,
					Content:   string(respJSON),
				})
			}
		}

		if len(blocks) == 1 && blocks[0].Type == "text" {
			claudeMsg.Content = blocks[0].Text
		} else if len(blocks) > 0 {
			claudeMsg.Content = blocks
		}

		claudeReq.Messages = append(claudeReq.Messages, claudeMsg)
	}

	// Convert tools
	for _, tool := range req.Tools {
		for _, decl := range tool.FunctionDeclarations {
			claudeReq.Tools = append(claudeReq.Tools, ClaudeTool{
				Name:        decl.Name,
				Description: decl.Description,
				InputSchema: decl.Parameters,
			})
		}
	}

	return json.Marshal(claudeReq)
}
