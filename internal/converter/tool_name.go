package converter

import (
	"fmt"
	"hash/crc32"
)

const maxToolNameLen = 64

func shortenNameIfNeeded(name string) string {
	if len(name) <= maxToolNameLen {
		return name
	}
	hash := crc32.ChecksumIEEE([]byte(name))
	// Keep a stable prefix to preserve readability, add hash suffix for uniqueness.
	prefixLen := maxToolNameLen - 9 // "_" + 8 hex
	return fmt.Sprintf("%s_%08x", name[:prefixLen], hash)
}

func buildShortNameMap(names []string) map[string]string {
	result := make(map[string]string, len(names))
	used := make(map[string]int)
	for _, name := range names {
		short := shortenNameIfNeeded(name)
		if count, ok := used[short]; ok {
			count++
			used[short] = count
			short = shortenNameIfNeeded(fmt.Sprintf("%s_%d", name, count))
		} else {
			used[short] = 0
		}
		result[name] = short
	}
	return result
}
