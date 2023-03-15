package assert

import (
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type CheckFunc func(t test.TestHelper, input string)

func OutputContains(str string, successMsg, failureMsg string) CheckFunc {
	return func(t test.TestHelper, input string) {
		t.T().Helper()
		common.CheckOutputContains(t, input, str, successMsg, failureMsg, assertFailure)
	}
}

func OutputDoesNotContain(str string, successMsg, failureMsg string) CheckFunc {
	return func(t test.TestHelper, input string) {
		t.T().Helper()
		common.CheckOutputDoesNotContain(t, input, str, successMsg, failureMsg, assertFailure)
	}
}
