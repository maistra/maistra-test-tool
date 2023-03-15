package shell

import (
	"fmt"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func ExecuteIgnoreError(t test.TestHelper, cmd string) {
	_, _ = util.Shell(cmd)
}

func Execute(t test.TestHelper, cmd string, checks ...assert.CheckFunc) string {
	t.T().Helper()
	output, err := util.Shell(cmd)
	if err != nil {
		t.Fatalf("Command failed: %q", cmd)
	}
	for _, check := range checks {
		check(t, output)
	}
	return output
}

func Executef(t test.TestHelper, format string, args ...any) string {
	t.T().Helper()
	return Execute(t, fmt.Sprintf(format, args...))
}
