package converter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCodexToGeminiRequestAndStream(t *testing.T) {
	req := CodexRequest{
		Instructions: "sys",
		Input: []interface{}{
			map[string]interface{}{"type": "message", "role": "user", "content": []interface{}{map[string]interface{}{"type": "input_text", "text": "hi"}}},
			map[string]interface{}{"type": "function_call", "name": "tool", "call_id": "call_1", "arguments": `{"x":1}`},
			map[string]interface{}{"type": "function_call_output", "call_id": "call_1", "output": "ok"},
		},
		Reasoning: &CodexReasoning{Effort: "auto"},
		Tools:     []CodexTool{{Type: "function", Name: "tool", Parameters: map[string]interface{}{"type": "object"}}},
	}
	body, _ := json.Marshal(req)
	conv := &codexToGeminiRequest{}
	out, err := conv.Transform(body, "gemini", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var geminiReq GeminiRequest
	if err := json.Unmarshal(out, &geminiReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if geminiReq.SystemInstruction == nil || len(geminiReq.Contents) == 0 {
		t.Fatalf("gemini request missing")
	}

	chunk := GeminiStreamChunk{Candidates: []GeminiCandidate{{
		Content: GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "hello"}, {FunctionCall: &GeminiFunctionCall{Name: "tool_call_1", Args: map[string]interface{}{"x": 1}}}}},
		Index:   0,
	}}}
	chunkBody, _ := json.Marshal(chunk)
	state := NewTransformState()
	respConv := &codexToGeminiResponse{}
	stream := append(FormatSSE("", json.RawMessage(chunkBody)), FormatDone()...)
	streamOut, err := respConv.TransformChunk(stream, state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(streamOut), "response.output_text.delta") {
		t.Fatalf("missing output_text delta")
	}
	if !strings.Contains(string(streamOut), "response.output_item.added") {
		t.Fatalf("missing output_item added")
	}
	if !strings.Contains(string(streamOut), "response.completed") {
		t.Fatalf("missing response.completed")
	}
}

func TestGeminiToCodexStreamCompletion(t *testing.T) {
	resp := CodexResponse{Usage: CodexUsage{InputTokens: 1, OutputTokens: 2, TotalTokens: 3}}
	event := CodexStreamEvent{Type: "response.completed", Response: &resp}
	body, _ := json.Marshal(event)
	state := NewTransformState()
	conv := &geminiToCodexResponse{}
	out, err := conv.TransformChunk(FormatSSE("", json.RawMessage(body)), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "finishReason") {
		t.Fatalf("missing finishReason")
	}
}

func TestGeminiToCodexStreamEvents(t *testing.T) {
	state := NewTransformState()
	created := CodexStreamEvent{Type: "response.created", Response: &CodexResponse{ID: "resp_1"}}
	createdBody, _ := json.Marshal(created)
	text := CodexStreamEvent{Type: "response.output_text.delta", Delta: &CodexDelta{Type: "output_text_delta", Text: "hi"}}
	textBody, _ := json.Marshal(text)
	item := CodexStreamEvent{Type: "response.output_item.added", Item: &CodexOutput{Type: "function_call", Name: "tool", CallID: "call_1", Arguments: `{"x":1}`}}
	itemBody, _ := json.Marshal(item)
	completed := CodexStreamEvent{Type: "response.completed", Response: &CodexResponse{Usage: CodexUsage{InputTokens: 1, OutputTokens: 2, TotalTokens: 3}}}
	completedBody, _ := json.Marshal(completed)

	stream := append(FormatSSE("", json.RawMessage(createdBody)), FormatSSE("", json.RawMessage(textBody))...)
	stream = append(stream, FormatSSE("", json.RawMessage(itemBody))...)
	stream = append(stream, FormatSSE("", json.RawMessage(completedBody))...)

	conv := &geminiToCodexResponse{}
	out, err := conv.TransformChunk(stream, state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "functionCall") {
		t.Fatalf("missing functionCall")
	}
	if !strings.Contains(string(out), "finishReason") {
		t.Fatalf("missing finishReason")
	}
}

func TestGeminiToCodexStreamInvalidJSON(t *testing.T) {
	state := NewTransformState()
	conv := &geminiToCodexResponse{}
	out, err := conv.TransformChunk(FormatSSE("", "\"oops\""), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no output")
	}
}

func TestCodexToGeminiStreamInvalidJSON(t *testing.T) {
	state := NewTransformState()
	conv := &codexToGeminiResponse{}
	out, err := conv.TransformChunk(FormatSSE("", "\"oops\""), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no output")
	}
}

func TestGeminiToCodexStreamDone(t *testing.T) {
	state := NewTransformState()
	conv := &geminiToCodexResponse{}
	out, err := conv.TransformChunk(FormatDone(), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no output on done")
	}
}
