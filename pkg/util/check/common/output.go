package common

import (
	"fmt"
	"strings"

	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type CheckFunc func(t test.TestHelper, input string)

func CheckOutputContains(t test.TestHelper, output, str, successMsg, failureMsg string, failure FailureFunc) {
	t.T().Helper()
	if strings.Contains(output, str) {
		if successMsg == "" {
			successMsg = fmt.Sprintf("string '%s' found in output", str)
		}
		logSuccess(t, successMsg)
	} else {
		detailMsg := fmt.Sprintf("expected to find the string '%s' in the command output, but it wasn't found", str)
		if !t.WillRetry() {
			detailMsg += "\nfull output:\n" + output
		}
		failure(t, failureMsg, detailMsg)
	}
}

func CheckOutputDoesNotContain(t test.TestHelper, output, str, successMsg, failureMsg string, failure FailureFunc) {
	t.T().Helper()
	if strings.Contains(output, str) {
		detailMsg := fmt.Sprintf("expected the string '%s' to be absent from the command output, but it was present", str)
		if !t.WillRetry() {
			detailMsg += "\nfull output:\n" + output
		}
		failure(t, failureMsg, detailMsg)
	} else {
		if successMsg == "" {
			successMsg = fmt.Sprintf("string '%s' not found in output", str)
		}
		logSuccess(t, successMsg)
	}
}
