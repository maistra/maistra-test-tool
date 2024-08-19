// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package heredoc

import (
	"fmt"
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

func Docf(indentedFormat string, a ...any) string {
	return Doc(fmt.Sprintf(indentedFormat, a...))
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
