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
