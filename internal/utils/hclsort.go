package utils

import (
	"regexp"
	"sort"
	"strings"
)

type NamedHCL struct {
	Name string
	HCL  string
}

// JoinHCLBlocksSorted sorts by Name (or parses from HCL) and joins.
func JoinHCLBlocksSorted(blocks []NamedHCL) string {
	re := regexp.MustCompile(`(?m)^\s*resource\s+"[^"]+"\s+"([^"]+)"`)
	sort.Slice(blocks, func(i, j int) bool {
		nameI := blocks[i].Name
		nameJ := blocks[j].Name
		if nameI == "" {
			if m := re.FindStringSubmatch(blocks[i].HCL); len(m) > 1 {
				nameI = m[1]
			}
		}
		if nameJ == "" {
			if m := re.FindStringSubmatch(blocks[j].HCL); len(m) > 1 {
				nameJ = m[1]
			}
		}
		return strings.ToLower(nameI) < strings.ToLower(nameJ)
	})
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		// Normalize trailing whitespace to avoid extra blank lines
		trimmed := strings.TrimRightFunc(b.HCL, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n' || r == '\r'
		})
		parts = append(parts, trimmed)
	}
	// Join with exactly two newlines between resources
	return strings.Join(parts, "\n\n")
}

// SortAllResourceBlocks sorts all resource blocks by name, preserving header/suffix.
func SortAllResourceBlocks(hcl string) string {
	// Robust block splitter using brace counting to avoid truncating nested blocks
	type blockIdx struct{ start, end int }

	// Find indices of resource starts
	reStart := regexp.MustCompile(`(?m)^\s*resource\s+"[^"]+"\s+"[^"]+"\s*\{`)
	locs := reStart.FindAllStringIndex(hcl, -1)
	if len(locs) == 0 {
		return hcl
	}

	// Scan to find end of each block by balancing braces
	blocks := make([]blockIdx, 0, len(locs))
	for _, loc := range locs {
		start := loc[0]
		// brace count starts at 1 for the opening '{' matched
		braceCount := 1
		inString := false
		esc := false
		i := loc[1]
		for i < len(hcl) {
			ch := hcl[i]
			if inString {
				if esc {
					esc = false
				} else {
					if ch == '\\' {
						esc = true
					} else if ch == '"' {
						inString = false
					}
				}
			} else {
				if ch == '"' {
					inString = true
				} else if ch == '{' {
					braceCount++
				} else if ch == '}' {
					braceCount--
					if braceCount == 0 {
						// include following whitespace/newlines
						end := i + 1
						// consume trailing spaces/newlines
						for end < len(hcl) {
							c := hcl[end]
							if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
								end++
							} else {
								break
							}
						}
						blocks = append(blocks, blockIdx{start: start, end: end})
						break
					}
				}
			}
			i++
		}
	}

	// Prefix before first resource
	prefix := hcl[:blocks[0].start]
	named := make([]NamedHCL, 0, len(blocks))
	reName := regexp.MustCompile(`(?m)^\s*resource\s+"[^"]+"\s+"([^"]+)"`)
	for _, b := range blocks {
		blk := hcl[b.start:b.end]
		name := ""
		if m := reName.FindStringSubmatch(blk); len(m) > 1 {
			name = m[1]
		}
		// Normalize block to avoid carrying trailing blank lines
		normalized := strings.TrimRightFunc(blk, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n' || r == '\r'
		})
		named = append(named, NamedHCL{Name: name, HCL: normalized})
	}
	sorted := JoinHCLBlocksSorted(named)
	suffix := hcl[blocks[len(blocks)-1].end:]
	if !strings.HasSuffix(prefix, "\n") {
		prefix += "\n"
	}
	return prefix + sorted + suffix
}
