package require

import (
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func OutputContains(str string, successMsg, failureMsg string) common.CheckFunc {
	return func(t test.TestHelper, input string) {
		t.T().Helper()
		common.CheckOutputContains(t, input, str, successMsg, failureMsg, requireFailure)
	}
}

func OutputDoesNotContain(str string, successMsg, failureMsg string) common.CheckFunc {
	return func(t test.TestHelper, input string) {
		t.T().Helper()
		common.CheckOutputDoesNotContain(t, input, str, successMsg, failureMsg, requireFailure)
	}
}
