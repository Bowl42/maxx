package converter

// Exported wrappers for converter helpers used by subpackages.
func StringifyContent(content interface{}) string {
	return stringifyContent(content)
}

func ShortenNameIfNeeded(name string) string {
	return shortenNameIfNeeded(name)
}

func BuildShortNameMap(names []string) map[string]string {
	return buildShortNameMap(names)
}
