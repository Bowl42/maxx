package converter

import (
	"strconv"
	"strings"
)

const maxToolNameLen = 64

func shortenNameIfNeeded(name string) string {
	if len(name) <= maxToolNameLen {
		return name
	}
	if strings.HasPrefix(name, "mcp__") {
		idx := strings.LastIndex(name, "__")
		if idx > 0 {
			candidate := "mcp__" + name[idx+2:]
			if len(candidate) > maxToolNameLen {
				return candidate[:maxToolNameLen]
			}
			return candidate
		}
	}
	return name[:maxToolNameLen]
}

func buildShortNameMap(names []string) map[string]string {
	used := map[string]struct{}{}
	result := make(map[string]string, len(names))

	baseCandidate := func(n string) string {
		if len(n) <= maxToolNameLen {
			return n
		}
		if strings.HasPrefix(n, "mcp__") {
			idx := strings.LastIndex(n, "__")
			if idx > 0 {
				cand := "mcp__" + n[idx+2:]
				if len(cand) > maxToolNameLen {
					cand = cand[:maxToolNameLen]
				}
				return cand
			}
		}
		return n[:maxToolNameLen]
	}

	makeUnique := func(cand string) string {
		if _, ok := used[cand]; !ok {
			return cand
		}
		base := cand
		for i := 1; ; i++ {
			suffix := "_" + strconv.Itoa(i)
			allowed := maxToolNameLen - len(suffix)
			if allowed < 0 {
				allowed = 0
			}
			tmp := base
			if len(tmp) > allowed {
				tmp = tmp[:allowed]
			}
			tmp = tmp + suffix
			if _, ok := used[tmp]; !ok {
				return tmp
			}
		}
	}

	for _, n := range names {
		cand := baseCandidate(n)
		uniq := makeUnique(cand)
		used[uniq] = struct{}{}
		result[n] = uniq
	}
	return result
}
