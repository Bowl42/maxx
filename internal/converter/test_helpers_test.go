package converter

func codexInputHasRoleText(input interface{}, role string, text string) bool {
	items, ok := input.([]interface{})
	if !ok {
		return false
	}
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok || m["type"] != "message" || m["role"] != role {
			continue
		}
		switch content := m["content"].(type) {
		case string:
			if content == text {
				return true
			}
		case []interface{}:
			for _, part := range content {
				pm, ok := part.(map[string]interface{})
				if ok && pm["text"] == text {
					return true
				}
			}
		}
	}
	return false
}
