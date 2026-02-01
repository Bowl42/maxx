package converter

import "strings"

// remapFunctionCallArgs remaps Gemini's function call arguments to Claude Code expected format
// This is critical for Claude Code compatibility as Gemini sometimes uses different parameter names
func remapFunctionCallArgs(toolName string, args map[string]interface{}) {
	if args == nil {
		return
	}

	toolNameLower := strings.ToLower(toolName)

	switch toolNameLower {
	case "grep":
		// Gemini uses "query", Claude Code expects "pattern"
		if query, ok := args["query"]; ok {
			if _, hasPattern := args["pattern"]; !hasPattern {
				args["pattern"] = query
				delete(args, "query")
			}
		}
		// Claude Code uses "path" (string), NOT "paths" (array)
		if _, hasPath := args["path"]; !hasPath {
			if paths, ok := args["paths"]; ok {
				pathStr := extractFirstPath(paths)
				args["path"] = pathStr
				delete(args, "paths")
			} else {
				args["path"] = "."
			}
		}

	case "glob":
		// Gemini uses "query", Claude Code expects "pattern"
		if query, ok := args["query"]; ok {
			if _, hasPattern := args["pattern"]; !hasPattern {
				args["pattern"] = query
				delete(args, "query")
			}
		}
		// Claude Code uses "path" (string), NOT "paths" (array)
		if _, hasPath := args["path"]; !hasPath {
			if paths, ok := args["paths"]; ok {
				pathStr := extractFirstPath(paths)
				args["path"] = pathStr
				delete(args, "paths")
			} else {
				args["path"] = "."
			}
		}

	case "read":
		// Gemini might use "path" vs "file_path"
		if path, ok := args["path"]; ok {
			if _, hasFilePath := args["file_path"]; !hasFilePath {
				args["file_path"] = path
				delete(args, "path")
			}
		}

	case "ls":
		// LS tool: ensure "path" parameter exists
		if _, hasPath := args["path"]; !hasPath {
			args["path"] = "."
		}
	}
}

// extractFirstPath extracts the first path from various input formats
func extractFirstPath(paths interface{}) string {
	switch v := paths.(type) {
	case []interface{}:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}
		return "."
	case string:
		return v
	default:
		return "."
	}
}
