package common

import (
	"fmt"
	"strings"

	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func CheckOutputContains(t test.TestHelper, output, str, successMsg, failureMsg string, failure FailureFunc) {
	t.T().Helper()
	if strings.Contains(output, str) {
		if successMsg == "" {
			successMsg = fmt.Sprintf("found %q in response", str)
		}
		logSuccess(t, successMsg)
	} else {
		detailMsg := fmt.Sprintf("expected command output to contain `%s`, but the output was:\n%v", str, output)
		failure(t, failureMsg, detailMsg)
	}
}

func CheckOutputDoesNotContain(t test.TestHelper, output, str, successMsg, failureMsg string, failure FailureFunc) {
	t.T().Helper()
	if strings.Contains(output, str) {
		detailMsg := fmt.Sprintf("expected command output to not contain `%s`, but the output was:\n%v", str, output)
		failure(t, failureMsg, detailMsg)
	} else {
		if successMsg == "" {
			successMsg = fmt.Sprintf("did not find %q in response", str)
		}
		logSuccess(t, successMsg)
	}
}
