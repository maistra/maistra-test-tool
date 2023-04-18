// Copyright 2021 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

// CreateTempfile creates a tempfile string.
func CreateTempfile(tmpDir, prefix, suffix string) (string, error) {
	f, err := os.CreateTemp(tmpDir, prefix)
	if err != nil {
		return "", err
	}
	var tmpName string
	if tmpName, err = filepath.Abs(f.Name()); err != nil {
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	if err = os.Remove(tmpName); err != nil {
		log.Log.Errorf("CreateTempfile unable to remove %s", tmpName)
		return "", err
	}
	return tmpName + suffix, nil
}

// WriteTempfile creates a tempfile with the specified contents.
func WriteTempfile(tmpDir, prefix, suffix, contents string) (string, error) {
	fname, err := CreateTempfile(tmpDir, prefix, suffix)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(fname, []byte(contents), 0644); err != nil {
		return "", err
	}
	return fname, nil
}

// Shell runs command on shell and get back output and error if get one
func Shell(format string, args ...interface{}) (string, error) {
	return sh(context.Background(), format, true, true, true, "", args...)
}

// ShellWithInput runs command on shell, passing in the specified reader as the stdin, and get back output and error if get one
func ShellWithInput(input string, format string, args ...interface{}) (string, error) {
	return sh(context.Background(), format, true, true, true, input, args...)
}

// ShellMuteOutputError run command on shell and get back output and error if get one
// without logging the output or errors
func ShellMuteOutputError(format string, args ...interface{}) (string, error) {
	return sh(context.Background(), format, true, false, false, "", args...)
}

// ShellSilent runs command on shell and get back output and error if get one
// without logging the command or output.
func ShellSilent(format string, args ...interface{}) (string, error) {
	return sh(context.Background(), format, false, false, false, "", args...)
}

func sh(ctx context.Context, format string, logCommand, logOutput, logError bool, input string, args ...interface{}) (string, error) {
	command := fmt.Sprintf(format, args...)
	if logCommand {
		log.Log.Infof("Running command: %s", command)
	}
	c := exec.CommandContext(ctx, "sh", "-c", command) // #nosec
	if input != "" {
		log.Log.Infof("Command input:\n%s", input)
		c.Stdin = strings.NewReader(input)
	}
	bytes, err := c.CombinedOutput()
	if logOutput {
		if output := strings.TrimSuffix(string(bytes), "\n"); len(output) > 0 {
			log.Log.Infof("Command output: \n%s", output)
		}
	}

	if err != nil {
		if logError {
			log.Log.Infof("Command error: %v", err)
		}
		return string(bytes), fmt.Errorf("command failed: %s", string(bytes))
	}
	return string(bytes), nil
}
