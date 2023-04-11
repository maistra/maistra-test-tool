package shell

import (
	"fmt"
	"os"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func ExecuteIgnoreError(t test.TestHelper, cmd string) {
	_, _ = util.Shell(cmd)
}

func Execute(t test.TestHelper, cmd string, checks ...common.CheckFunc) string {
	t.T().Helper()
	output, err := util.Shell(cmd)
	if err != nil {
		t.Fatalf("Command failed: %q\nError: %s", cmd, err)
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

func CreateTempDir(t test.TestHelper, namePrefix string) string {
	dir, err := os.MkdirTemp("/tmp", namePrefix)
	if err != nil {
		t.Fatalf("could not create temp dir %s: %v", namePrefix, err)
	}
	return dir
}
