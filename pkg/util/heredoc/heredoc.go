package heredoc

import (
	"math"
	"strings"
)

func Doc(indented string) string {
	lines := strings.Split(indented, "\n")
	if len(lines) > 0 && len(lines[0]) == 0 {
		lines = lines[1:]
	}
	baseIndent := getBaseIndent(lines)
	lines = removeBaseIndent(lines, baseIndent)
	return strings.Join(lines, "\n")
}

func getBaseIndent(lines []string) int {
	baseIndent := math.MaxInt
	for _, line := range lines {
		indent := 0
		for _, r := range line {
			if r == ' ' || r == '\t' {
				indent++
			} else {
				break
			}
		}

		if indent < baseIndent {
			baseIndent = indent
		}
	}
	return baseIndent
}

func removeBaseIndent(lines []string, baseIndent int) []string {
	for i, line := range lines {
		if len(line) > baseIndent {
			lines[i] = line[baseIndent:]
		}
	}
	return lines
}
