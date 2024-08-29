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

package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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

func ExecuteWithInput(t test.TestHelper, cmd string, input string, checks ...common.CheckFunc) string {
	t.T().Helper()
	return ExecuteWithEnvAndInput(t, nil, cmd, input, checks...)
}

func ExecuteWithEnv(t test.TestHelper, env []string, cmd string, checks ...common.CheckFunc) string {
	t.T().Helper()
	return ExecuteWithEnvAndInput(t, env, cmd, "", checks...)
}

func ExecuteWithEnvAndInput(t test.TestHelper, env []string, cmd string, input string, checks ...common.CheckFunc) string {
	t.T().Helper()
	output, err := execShellCommand(cmd, env, input)
	if err != nil {
		t.Fatalf("Command failed: %s\n%serror: %s", cmd, appendNewLine(output), err)
	}
	for _, check := range checks {
		check(t, output)
	}
	return output
}

func appendNewLine(str string) string {
	if str == "" {
		return ""
	}
	if strings.HasSuffix(str, "\n") {
		return str
	}
	return str + "\n"
}

func execShellCommand(command string, env []string, input string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Env = env
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

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
