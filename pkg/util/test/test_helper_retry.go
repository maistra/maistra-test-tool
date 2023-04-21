package test

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
)

const retryPanicKey = "RetryTestHelper.FailNow"

func NewRetryTestHelper(t *testing.T, currentStep, attempt, maxAttempts int) *RetryTestHelper {
	return &RetryTestHelper{
		testHelper: testHelper{
			t:           t,
			currentStep: currentStep,
		},
		attempt:     attempt,
		maxAttempts: maxAttempts,
	}
}

type RetryTestHelper struct {
	testHelper
	failed      bool
	attempt     int
	maxAttempts int

	logBuffer []string
}

var _ TestHelper = &RetryTestHelper{}

func (t *RetryTestHelper) Log(args ...any) {
	t.t.Helper()
	t.logOrAppendToBuffer(fmt.Sprint(args...))
}

func (t *RetryTestHelper) Logf(format string, args ...any) {
	t.t.Helper()
	t.logOrAppendToBuffer(fmt.Sprintf(format, args...))
}

func (t *RetryTestHelper) LogSuccess(str string) {
	t.t.Helper()
	t.Log(SuccessPrefix + str)
}

func (t *RetryTestHelper) logOrAppendToBuffer(str string) {
	t.t.Helper()
	if env.IsLogFailedRetryAttempts() {
		t.t.Log(t.indent() + str)
	} else {
		t.logBuffer = append(t.logBuffer, str)
	}
}

func (t *RetryTestHelper) FlushLogBuffer() {
	t.t.Helper()
	if !env.IsLogFailedRetryAttempts() {
		for _, s := range t.logBuffer {
			t.t.Log(t.indent() + s)
		}
		t.logBuffer = nil
	}
}

func (t *RetryTestHelper) Fail() {
	t.failed = true
}

func (t *RetryTestHelper) FailNow() {
	t.Fail()
	panic(retryPanicKey)
}

func (t *RetryTestHelper) Failed() bool {
	return t.failed
}

func (t *RetryTestHelper) Error(args ...any) {
	t.t.Helper()
	t.Log(FailurePrefix + fmt.Sprint(args...))
	t.Fail()
}

func (t *RetryTestHelper) Errorf(format string, args ...any) {
	t.t.Helper()
	t.Error(fmt.Sprintf(format, args...))
}

func (t *RetryTestHelper) Fatal(args ...any) {
	t.t.Helper()
	t.Error(args...)
	t.FailNow()
}

func (t *RetryTestHelper) Fatalf(format string, args ...any) {
	t.t.Helper()
	t.Fatal(fmt.Sprintf(format, args...))
}

func (t *RetryTestHelper) WillRetry() bool {
	return t.attempt < t.maxAttempts-1
}

func (t *RetryTestHelper) Attempt(f func(t TestHelper)) {
	t.T().Helper()
	defer func() {
		// recover from panic thrown in RetryTestHelper.FailNow() to prevent attempt from continuing
		if err := recover(); err != nil {
			if err == retryPanicKey {
				t.Fail()
			} else {
				panic(err)
			}
		}
	}()
	f(t)
}
