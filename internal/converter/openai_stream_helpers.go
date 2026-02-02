package converter

import (
	"encoding/json"
	"strconv"
	"time"
)

func ensureOpenAIStreamMeta(state *TransformState) (string, string, int64) {
	id := "chatcmpl-unknown"
	if state != nil && state.MessageID != "" {
		id = state.MessageID
	}
	model := "unknown"
	if state != nil && state.Model != "" {
		model = state.Model
	}
	created := int64(0)
	if state != nil {
		created = state.CreatedAt
	}
	if created == 0 {
		created = time.Now().Unix()
		if state != nil {
			state.CreatedAt = created
		}
	}
	return id, model, created
}

func formatOpenAIStreamChunk(state *TransformState, delta map[string]interface{}, finishReason *string) []byte {
	if delta == nil {
		delta = map[string]interface{}{}
	}
	id, model, created := ensureOpenAIStreamMeta(state)
	choice := map[string]interface{}{
		"index":         0,
		"delta":         delta,
		"finish_reason": nil,
	}
	if finishReason != nil {
		choice["finish_reason"] = *finishReason
	}
	chunk := map[string]interface{}{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []interface{}{choice},
	}
	return FormatSSE("", chunk)
}

func parseUnixSeconds(v interface{}) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return n
		}
	case string:
		if n, err := strconv.ParseInt(t, 10, 64); err == nil {
			return n
		}
	}
	return 0
}
