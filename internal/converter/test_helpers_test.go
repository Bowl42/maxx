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
		if content, ok := m["content"].(string); ok && content == text {
			return true
		}
	}
	return false
}
