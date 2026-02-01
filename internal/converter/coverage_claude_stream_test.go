package converter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGeminiToClaudeRequestAndStream(t *testing.T) {
	req := GeminiRequest{
		SystemInstruction: &GeminiContent{Parts: []GeminiPart{{Text: "sys"}}},
		Contents: []GeminiContent{{
			Role: "user",
			Parts: []GeminiPart{{Text: "hi"}, {
				FunctionCall: &GeminiFunctionCall{Name: "tool", Args: map[string]interface{}{"x": 1}},
			}, {
				FunctionResponse: &GeminiFunctionResponse{Name: "tool", Response: map[string]interface{}{"result": "ok"}},
			}},
		}},
	}
	body, _ := json.Marshal(req)
	conv := &geminiToClaudeRequest{}
	out, err := conv.Transform(body, "claude", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var claudeReq ClaudeRequest
	if err := json.Unmarshal(out, &claudeReq); err != nil {
		t.Fatalf("unmarshal claude: %v", err)
	}
	if claudeReq.System != "sys" {
		t.Fatalf("system mismatch")
	}
	if len(claudeReq.Messages) == 0 {
		t.Fatalf("messages missing")
	}

	chunk := GeminiStreamChunk{
		Candidates: []GeminiCandidate{{
			Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "t", Thought: true}, {Text: "hello"}}},
			FinishReason: "STOP",
			Index:        0,
		}},
	}
	chunkBody, _ := json.Marshal(chunk)
	state := NewTransformState()
	respConv := &geminiToClaudeResponse{}
	streamOut, err := respConv.TransformChunk(append(FormatSSE("", json.RawMessage(chunkBody)), FormatDone()...), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(streamOut), "thinking_delta") {
		t.Fatalf("missing thinking delta")
	}
}

func TestClaudeToCodexResponseAndStream(t *testing.T) {
	resp := ClaudeResponse{
		ID:    "msg_1",
		Model: "claude",
		Usage: ClaudeUsage{InputTokens: 1, OutputTokens: 2},
		Content: []ClaudeContentBlock{{
			Type: "text",
			Text: "hello",
		}, {
			Type:  "tool_use",
			ID:    "call_1",
			Name:  "tool",
			Input: map[string]interface{}{"x": 1},
		}},
	}
	body, _ := json.Marshal(resp)
	conv := &claudeToCodexResponse{}
	out, err := conv.Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexResp CodexResponse
	if err := json.Unmarshal(out, &codexResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(codexResp.Output) < 2 {
		t.Fatalf("codex output missing")
	}

	state := NewTransformState()
	start := ClaudeStreamEvent{Type: "message_start", Message: &ClaudeResponse{ID: "msg_1"}}
	startBody, _ := json.Marshal(start)
	delta := ClaudeStreamEvent{Type: "content_block_delta", Delta: &ClaudeStreamDelta{Type: "text_delta", Text: "hi"}}
	deltaBody, _ := json.Marshal(delta)
	stop := ClaudeStreamEvent{Type: "message_stop"}
	stopBody, _ := json.Marshal(stop)
	stream := append(FormatSSE("", json.RawMessage(startBody)), FormatSSE("", json.RawMessage(deltaBody))...)
	stream = append(stream, FormatSSE("", json.RawMessage(stopBody))...)
	stream = append(stream, FormatDone()...)

	streamOut, err := conv.TransformChunk(stream, state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(streamOut), "response.created") {
		t.Fatalf("missing response.created")
	}
	if !strings.Contains(string(streamOut), "response.output_item.delta") {
		t.Fatalf("missing delta")
	}
	if !strings.Contains(string(streamOut), "response.done") {
		t.Fatalf("missing response.done")
	}
}

func TestClaudeToGeminiStream(t *testing.T) {
	state := NewTransformState()
	delta := ClaudeStreamEvent{Type: "content_block_delta", Delta: &ClaudeStreamDelta{Type: "text_delta", Text: "hi"}}
	deltaBody, _ := json.Marshal(delta)
	msgDelta := ClaudeStreamEvent{Type: "message_delta", Usage: &ClaudeUsage{OutputTokens: 2}}
	msgDeltaBody, _ := json.Marshal(msgDelta)
	stop := ClaudeStreamEvent{Type: "message_stop"}
	stopBody, _ := json.Marshal(stop)
	stream := append(FormatSSE("", json.RawMessage(deltaBody)), FormatSSE("", json.RawMessage(msgDeltaBody))...)
	stream = append(stream, FormatSSE("", json.RawMessage(stopBody))...)

	conv := &claudeToGeminiResponse{}
	out, err := conv.TransformChunk(stream, state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "finishReason") {
		t.Fatalf("missing finishReason")
	}
}

func TestGeminiToClaudeStreamFunctionCall(t *testing.T) {
	state := NewTransformState()
	chunk := GeminiStreamChunk{Candidates: []GeminiCandidate{{
		Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{FunctionCall: &GeminiFunctionCall{Name: "tool", Args: map[string]interface{}{"x": 1}}}}},
		FinishReason: "STOP",
		Index:        0,
	}}}
	body, _ := json.Marshal(chunk)
	conv := &geminiToClaudeResponse{}
	out, err := conv.TransformChunk(FormatSSE("", json.RawMessage(body)), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "tool_use") {
		t.Fatalf("missing tool_use")
	}
}

func TestGeminiToClaudeStreamMaxTokens(t *testing.T) {
	state := NewTransformState()
	chunk := GeminiStreamChunk{Candidates: []GeminiCandidate{{
		Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "hi"}}},
		FinishReason: "MAX_TOKENS",
		Index:        0,
	}}}
	body, _ := json.Marshal(chunk)
	conv := &geminiToClaudeResponse{}
	out, err := conv.TransformChunk(FormatSSE("", json.RawMessage(body)), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "max_tokens") {
		t.Fatalf("expected max_tokens")
	}
}

func TestGeminiToClaudeStreamUsageMetadata(t *testing.T) {
	state := NewTransformState()
	chunk := GeminiStreamChunk{UsageMetadata: &GeminiUsageMetadata{PromptTokenCount: 1, CandidatesTokenCount: 2}, Candidates: []GeminiCandidate{{
		Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "hi"}}},
		FinishReason: "STOP",
		Index:        0,
	}}}
	body, _ := json.Marshal(chunk)
	conv := &geminiToClaudeResponse{}
	out, err := conv.TransformChunk(FormatSSE("", json.RawMessage(body)), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "output_tokens") {
		t.Fatalf("expected usage tokens")
	}
}

func TestGeminiToClaudeStreamFunctionCallOnly(t *testing.T) {
	state := NewTransformState()
	chunk := GeminiStreamChunk{Candidates: []GeminiCandidate{{
		Content: GeminiContent{Role: "model", Parts: []GeminiPart{{FunctionCall: &GeminiFunctionCall{Name: "tool", Args: map[string]interface{}{"x": 1}}}}},
		Index:   0,
	}}}
	body, _ := json.Marshal(chunk)
	conv := &geminiToClaudeResponse{}
	out, err := conv.TransformChunk(FormatSSE("", json.RawMessage(body)), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "tool_use") {
		t.Fatalf("expected tool_use")
	}
}

func TestClaudeToGeminiStreamUsage(t *testing.T) {
	state := NewTransformState()
	msgDelta := ClaudeStreamEvent{Type: "message_delta", Usage: &ClaudeUsage{OutputTokens: 2}}
	msgDeltaBody, _ := json.Marshal(msgDelta)
	stop := ClaudeStreamEvent{Type: "message_stop"}
	stopBody, _ := json.Marshal(stop)
	stream := append(FormatSSE("", json.RawMessage(msgDeltaBody)), FormatSSE("", json.RawMessage(stopBody))...)
	conv := &claudeToGeminiResponse{}
	out, err := conv.TransformChunk(stream, state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "usageMetadata") {
		t.Fatalf("expected usageMetadata")
	}
}

func TestGeminiToClaudeStreamThoughtAndText(t *testing.T) {
	state := NewTransformState()
	chunk := GeminiStreamChunk{Candidates: []GeminiCandidate{{
		Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{Thought: true, Text: "t"}, {Text: "hi"}}},
		FinishReason: "STOP",
		Index:        0,
	}}}
	body, _ := json.Marshal(chunk)
	conv := &geminiToClaudeResponse{}
	out, err := conv.TransformChunk(FormatSSE("", json.RawMessage(body)), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "thinking_delta") {
		t.Fatalf("expected thinking_delta")
	}
	if !strings.Contains(string(out), "text_delta") {
		t.Fatalf("expected text_delta")
	}
}

func TestClaudeToGeminiStreamDoneAndNonTextDelta(t *testing.T) {
	state := NewTransformState()
	delta := ClaudeStreamEvent{Type: "content_block_delta", Delta: &ClaudeStreamDelta{Type: "thinking_delta", Thinking: "t"}}
	deltaBody, _ := json.Marshal(delta)
	stream := append(FormatSSE("", json.RawMessage(deltaBody)), FormatDone()...)
	conv := &claudeToGeminiResponse{}
	out, err := conv.TransformChunk(stream, state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no output for non-text delta")
	}
}

func TestGeminiToClaudeStreamFinishStopsBlock(t *testing.T) {
	state := NewTransformState()
	chunk := GeminiStreamChunk{Candidates: []GeminiCandidate{{
		Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "hi"}}},
		FinishReason: "MAX_TOKENS",
		Index:        0,
	}}}
	body, _ := json.Marshal(chunk)
	conv := &geminiToClaudeResponse{}
	out, err := conv.TransformChunk(FormatSSE("", json.RawMessage(body)), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "max_tokens") {
		t.Fatalf("expected max_tokens stop reason")
	}
}

func TestGeminiToClaudeStreamMaxTokensStop(t *testing.T) {
	state := NewTransformState()
	chunk := GeminiStreamChunk{Candidates: []GeminiCandidate{{
		Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "hi"}}},
		FinishReason: "MAX_TOKENS",
		Index:        0,
	}}}
	body, _ := json.Marshal(chunk)
	conv := &geminiToClaudeResponse{}
	out, err := conv.TransformChunk(FormatSSE("", json.RawMessage(body)), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "max_tokens") {
		t.Fatalf("expected max_tokens")
	}
}

func TestGeminiToClaudeStreamThinkingThenTextStop(t *testing.T) {
	state := NewTransformState()
	chunk := GeminiStreamChunk{Candidates: []GeminiCandidate{{
		Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{Thought: true, Text: "t"}, {Text: "hi"}}},
		FinishReason: "STOP",
		Index:        0,
	}}}
	body, _ := json.Marshal(chunk)
	conv := &geminiToClaudeResponse{}
	out, err := conv.TransformChunk(FormatSSE("", json.RawMessage(body)), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "content_block_stop") {
		t.Fatalf("expected block stop")
	}
}

func TestGeminiToClaudeStreamTextThenThought(t *testing.T) {
	state := NewTransformState()
	chunk := GeminiStreamChunk{Candidates: []GeminiCandidate{{
		Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "hi"}, {Thought: true, Text: "t"}}},
		FinishReason: "STOP",
		Index:        0,
	}}}
	body, _ := json.Marshal(chunk)
	conv := &geminiToClaudeResponse{}
	out, err := conv.TransformChunk(FormatSSE("", json.RawMessage(body)), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "content_block_stop") {
		t.Fatalf("expected content_block_stop")
	}
}

func TestGeminiToClaudeStreamTextFinishLength(t *testing.T) {
	state := NewTransformState()
	chunk := GeminiStreamChunk{Candidates: []GeminiCandidate{{
		Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "hi"}}},
		FinishReason: "MAX_TOKENS",
		Index:        0,
	}}}
	body, _ := json.Marshal(chunk)
	conv := &geminiToClaudeResponse{}
	out, err := conv.TransformChunk(FormatSSE("", json.RawMessage(body)), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "max_tokens") {
		t.Fatalf("expected max_tokens")
	}
}

func TestGeminiToClaudeStreamFunctionCallAfterText(t *testing.T) {
	state := NewTransformState()
	chunk := GeminiStreamChunk{Candidates: []GeminiCandidate{{
		Content: GeminiContent{Role: "model", Parts: []GeminiPart{
			{Text: "hi"},
			{FunctionCall: &GeminiFunctionCall{Name: "tool", Args: map[string]interface{}{"x": 1}}},
		}},
	}}}
	body, _ := json.Marshal(chunk)
	conv := &geminiToClaudeResponse{}
	out, err := conv.TransformChunk(FormatSSE("", json.RawMessage(body)), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), "tool_use") || !strings.Contains(string(out), "content_block_stop") {
		t.Fatalf("expected tool call transition")
	}
}

func TestClaudeToCodexStreamInvalidJSON(t *testing.T) {
	state := NewTransformState()
	conv := &claudeToCodexResponse{}
	out, err := conv.TransformChunk(FormatSSE("", "\"oops\""), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no output")
	}
}

func TestCodexToClaudeStreamInvalidJSON(t *testing.T) {
	state := NewTransformState()
	conv := &codexToClaudeResponse{}
	out, err := conv.TransformChunk(FormatSSE("", "\"oops\""), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no output")
	}
}

func TestClaudeToGeminiStreamInvalidJSON(t *testing.T) {
	state := NewTransformState()
	conv := &claudeToGeminiResponse{}
	out, err := conv.TransformChunk(FormatSSE("", "\"oops\""), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no output")
	}
}
