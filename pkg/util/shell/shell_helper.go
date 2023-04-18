package shell

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func Executef(t test.TestHelper, format string, args ...any) string {
	t.T().Helper()
	return Execute(t, fmt.Sprintf(format, args...))
}

func Execute(t test.TestHelper, cmd string, checks ...common.CheckFunc) string {
	t.T().Helper()
	return ExecuteWithEnv(t, nil, cmd, checks...)
}

func ExecuteWithEnv(t test.TestHelper, env []string, cmd string, checks ...common.CheckFunc) string {
	t.T().Helper()
	output, err := execShellCommand(cmd, env)
	if err != nil {
		t.Fatalf("Command failed: %q\n%s\nError: %s", cmd, output, err)
	}
	for _, check := range checks {
		check(t, output)
	}
	return output
}

func execShellCommand(command string, env []string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Env = env
	bytes, err := cmd.CombinedOutput()
	return string(bytes), err
}

func CreateTempDir(t test.TestHelper, namePrefix string) string {
	dir, err := os.MkdirTemp("/tmp", namePrefix)
	if err != nil {
		t.Fatalf("could not create temp dir %s: %v", namePrefix, err)
	}
	return dir
}
