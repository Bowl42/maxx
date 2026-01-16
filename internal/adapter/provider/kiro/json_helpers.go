package kiro

import "encoding/json"

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
