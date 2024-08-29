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

import "testing"

func TestDoc(t *testing.T) {
	cases := []struct {
		input  string
		output string
	}{
		{
			input:  "no indentation",
			output: "no indentation",
		},
		{
			input:  "  two spaces",
			output: "two spaces",
		},
		{
			input:  "  three tabs",
			output: "three tabs",
		},
		{
			input:  "  one\n    two",
			output: "one\n  two",
		},
		{
			input:  "    two\n  one",
			output: "  two\none",
		},
		{
			input: `
               one
                 two`,
			output: "one\n  two",
		},
	}

	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			actual := Doc(tc.input)
			if actual != tc.output {
				t.Fatalf("Expected output to be:\n%s\nbut was:\n%s", tc.output, actual)
			}
		})
	}
}
