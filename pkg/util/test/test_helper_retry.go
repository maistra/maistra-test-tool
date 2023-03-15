package test

import (
	"fmt"
	"testing"
)

const retryPanicKey = "RetryTestHelper.FailNow"

func NewRetryTestHelper(t *testing.T, attempt, maxAttempts int) *RetryTestHelper {
	return &RetryTestHelper{
		testHelper: testHelper{
			t: t,
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
}

var _ TestHelper = &RetryTestHelper{}

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
	t.t.Log("transient error " + t.attemptString() + ": " + fmt.Sprint(args...))
	t.Fail()
}

func (t *RetryTestHelper) Errorf(format string, args ...any) {
	t.t.Helper()
	t.t.Logf("transient error "+t.attemptString()+": "+format, args...)
	t.Fail()
}

func (t *RetryTestHelper) Fatal(args ...any) {
	t.t.Helper()
	t.t.Log("transient error " + t.attemptString() + ": " + fmt.Sprint(args...))
	t.FailNow()
}

func (t *RetryTestHelper) Fatalf(format string, args ...any) {
	t.t.Helper()
	t.t.Logf("transient error "+t.attemptString()+": "+format, args...)
	t.FailNow()
}

// func (t *RetryTestHelper) LogStep(str string) {
// 	t.t.Helper()
// 	t.Log("")
// 	t.currentStep++
// 	t.Logf("STEP %d (attempt #%d): %s", t.currentStep, t.attempt+1, str)
// }

func (t *RetryTestHelper) attemptString() string {
	return "(will retry)"
	// return fmt.Sprintf("(will retry; attempt %d/%d)", t.attempt+1, t.maxAttempts)
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
