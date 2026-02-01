package converter

import (
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strings"
)

func stringifyContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var sb strings.Builder
		for _, part := range v {
			if m, ok := part.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					sb.WriteString(text)
				}
			}
		}
		return sb.String()
	default:
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
	}
	return ""
}

func parseInlineImage(url string) *GeminiInlineData {
	if !strings.HasPrefix(url, "data:") {
		return nil
	}
	parts := strings.SplitN(url[5:], ";base64,", 2)
	if len(parts) != 2 {
		return nil
	}
	if _, err := base64.StdEncoding.DecodeString(parts[1]); err != nil {
		return nil
	}
	return &GeminiInlineData{
		MimeType: parts[0],
		Data:     parts[1],
	}
}

func parseFilePart(part map[string]interface{}) *GeminiInlineData {
	fileObj, ok := part["file"].(map[string]interface{})
	if !ok {
		return nil
	}
	filename, _ := fileObj["filename"].(string)
	fileData, _ := fileObj["file_data"].(string)
	if filename == "" || fileData == "" {
		return nil
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	mime := mimeFromExt(ext)
	if mime == "" {
		return nil
	}
	return &GeminiInlineData{
		MimeType: mime,
		Data:     fileData,
	}
}

func mimeFromExt(ext string) string {
	switch ext {
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "pdf":
		return "application/pdf"
	case "txt":
		return "text/plain"
	case "json":
		return "application/json"
	case "csv":
		return "text/csv"
	}
	return ""
}

func parseToolChoice(choice interface{}) *GeminiToolConfig {
	if choice == nil {
		return nil
	}
	switch v := choice.(type) {
	case string:
		mode := strings.ToLower(strings.TrimSpace(v))
		switch mode {
		case "none":
			return &GeminiToolConfig{FunctionCallingConfig: &GeminiFunctionCallingConfig{Mode: "NONE"}}
		case "auto":
			return &GeminiToolConfig{FunctionCallingConfig: &GeminiFunctionCallingConfig{Mode: "AUTO"}}
		case "required", "any":
			return &GeminiToolConfig{FunctionCallingConfig: &GeminiFunctionCallingConfig{Mode: "ANY"}}
		}
	case map[string]interface{}:
		typ, _ := v["type"].(string)
		if typ == "function" {
			if fn, ok := v["function"].(map[string]interface{}); ok {
				if name, ok := fn["name"].(string); ok && name != "" {
					return &GeminiToolConfig{FunctionCallingConfig: &GeminiFunctionCallingConfig{
						Mode:                 "ANY",
						AllowedFunctionNames: []string{name},
					}}
				}
			}
		}
	}
	return nil
}
