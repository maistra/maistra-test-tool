package template

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"

	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var TemplateFuncMap = template.FuncMap{
	"toYaml": toYaml,
	"indent": indent,
	"until":  until,
	"image":  image,
}

func Run(t test.TestHelper, tmpl string, vars interface{}) string {
	t.T().Helper()
	tt, err := template.New("").
		Funcs(TemplateFuncMap).
		Parse(tmpl)
	if err != nil {
		t.Fatalf("could not execute template: %v:\n%s", err, addLineNumbers(tmpl))
	}
	var buf bytes.Buffer
	if err := tt.Execute(&buf, vars); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func addLineNumbers(str string) string {
	var builder strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(str))
	for i := 1; scanner.Scan(); i++ {
		lineNumStr := fmt.Sprintf("%3d", i)
		_, _ = fmt.Fprintf(&builder, "%s: %s\n", lineNumStr, scanner.Text())
	}
	return builder.String()
}

func indent(spaces int, source string) string {
	res := strings.Split(source, "\n")
	for i, line := range res {
		if i > 0 {
			res[i] = fmt.Sprintf(fmt.Sprintf("%% %ds%%s", spaces), "", line)
		}
	}
	return strings.Join(res, "\n")
}

func toYaml(value interface{}) string {
	y, err := yaml.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("Unable to marshal %v", value))
	}

	return string(y)
}

// Define an until function for template
func until(n int) []int {
	nums := make([]int, n)
	for i := 0; i < n; i++ {
		nums[i] = i
	}
	return nums
}
