package codexutil

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func NormalizeCodexInput(body []byte) []byte {
	input := gjson.GetBytes(body, "input")
	if !input.IsArray() {
		return body
	}

	for i, item := range input.Array() {
		itemType := item.Get("type").String()
		// Keep role for legacy Responses message items that omit "type" but still carry a valid "role".
		// Only strip role when the item explicitly declares a non-message type.
		if itemType != "" && itemType != "message" {
			if item.Get("role").Exists() {
				body, _ = sjson.DeleteBytes(body, fmt.Sprintf("input.%d.role", i))
			}
		}
		if itemType == "function_call" {
			id := strings.TrimSpace(item.Get("id").String())
			switch {
			case strings.HasPrefix(id, "fc_"):
			case id == "":
				body, _ = sjson.SetBytes(body, fmt.Sprintf("input.%d.id", i), "fc_"+uuid.NewString())
			default:
				body, _ = sjson.SetBytes(body, fmt.Sprintf("input.%d.id", i), "fc_"+id)
			}
		}
		if itemType == "function_call_output" {
			if !item.Get("output").Exists() {
				body, _ = sjson.SetBytes(body, fmt.Sprintf("input.%d.output", i), "")
			}
		}
	}

	return body
}
