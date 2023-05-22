package assert

import (
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func OutputContains(str string, successMsg, failureMsg string) common.CheckFunc {
	return func(t test.TestHelper, input string) {
		t.T().Helper()
		common.CheckOutputContainsAny(t, input, []string{str}, successMsg, failureMsg, assertFailure)
	}
}

func OutputContainsAny(str []string, successMsg, failureMsg string) common.CheckFunc {
	return func(t test.TestHelper, input string) {
		t.T().Helper()
		common.CheckOutputContainsAny(t, input, str, successMsg, failureMsg, assertFailure)
	}
}

func OutputDoesNotContain(str string, successMsg, failureMsg string) common.CheckFunc {
	return func(t test.TestHelper, input string) {
		t.T().Helper()
		common.CheckOutputDoesNotContain(t, input, str, successMsg, failureMsg, assertFailure)
	}
}

func CountExpectedString(str string, expectedOccurrenceNum int, successMsg, failureMsg string) common.CheckFunc {
	return func(t test.TestHelper, input string) {
		t.T().Helper()
		common.CountExpectedString(t, input, str, expectedOccurrenceNum, successMsg, failureMsg, assertFailure)
	}
}
