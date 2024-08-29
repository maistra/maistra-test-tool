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

package common

import (
	"fmt"
	"strings"

	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type CheckFunc func(t test.TestHelper, input string)

func CheckOutputContainsAny(t test.TestHelper, output string, expected []string, successMsg, failureMsg string, failure FailureFunc) {
	t.T().Helper()
	for _, str := range expected {
		if strings.Contains(output, str) {
			if successMsg == "" {
				successMsg = fmt.Sprintf("string '%s' found in output", str)
			}
			logSuccess(t, successMsg)
			return
		}
	}
	// none of the expected strings were found
	var detailMsg string
	if len(expected) == 1 {
		detailMsg = fmt.Sprintf("expected to find the string '%s' in the output, but it wasn't found", expected[0])
	} else {
		detailMsg = fmt.Sprintf("expected to find any of '%v' in the output, but none wasn't found", expected)
	}
	if !t.WillRetry() {
		detailMsg += "; full output:\n" + output
	}
	failure(t, failureMsg, detailMsg)
}

func CheckOutputDoesNotContain(t test.TestHelper, output, str, successMsg, failureMsg string, failure FailureFunc) {
	t.T().Helper()
	if strings.Contains(output, str) {
		detailMsg := fmt.Sprintf("expected the string '%s' to be absent from the command output, but it was present", str)
		if !t.WillRetry() {
			detailMsg += "; full output:\n" + output
		}
		failure(t, failureMsg, detailMsg)
	} else {
		if successMsg == "" {
			successMsg = fmt.Sprintf("string '%s' not found in output", str)
		}
		logSuccess(t, successMsg)
	}
}

func CountExpectedString(t test.TestHelper, output string, expected string, expectedOccurrenceNum int, successMsg, failureMsg string, failure FailureFunc) {
	t.T().Helper()
	if strings.Count(output, expected) != expectedOccurrenceNum {
		if successMsg == "" {
			successMsg = fmt.Sprintf("string '%s' found in output", expected)
		}
		logSuccess(t, successMsg)
		return
	}
	detailMsg := fmt.Sprintf("expected to find the string '%s' in the output, but it wasn't found", expected)
	if !t.WillRetry() {
		detailMsg += "; full output:\n" + output
	}
	failure(t, failureMsg, detailMsg)
}
