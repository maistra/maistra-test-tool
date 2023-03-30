package test

import (
	"fmt"
	"testing"
)

const flakyPanicKey = "flakyTestHelper.FailNow"

func NewFlakyTestHelper(t *testing.T) TestHelper {
	return &flakyTestHelper{
		testHelper: testHelper{
			t: t,
		},
	}
}

type flakyTestHelper struct {
	testHelper
	failed bool
}

var _ TestHelper = &flakyTestHelper{}

func (t *flakyTestHelper) Fail() {
	t.failed = true
}

func (t *flakyTestHelper) FailNow() {
	t.Fail()
	panic(flakyPanicKey)
}

func (t *flakyTestHelper) Failed() bool {
	return t.failed
}

func (t *flakyTestHelper) Error(args ...any) {
	t.t.Helper()
	t.Log("ERROR: " + fmt.Sprint(args...))
	t.Fail()
}

func (t *flakyTestHelper) Errorf(format string, args ...any) {
	t.t.Helper()
	t.Logf("ERROR: "+format, args...)
	t.Fail()
}

func (t *flakyTestHelper) Fatal(args ...any) {
	t.t.Helper()
	t.Log("FATAL: " + fmt.Sprint(args...))
	t.FailNow()
}

func (t *flakyTestHelper) Fatalf(format string, args ...any) {
	t.t.Helper()
	t.Logf("FATAL: "+format, args...)
	t.FailNow()
}

func (t *flakyTestHelper) Attempt(f func(t TestHelper)) {
	t.T().Helper()
	defer func() {
		// recover from panic thrown in flakyTestHelper.FailNow() to prevent attempt from continuing
		if err := recover(); err != nil {
			if err == flakyPanicKey {
				t.Fail()
			} else {
				panic(err)
			}
		}
	}()
	f(t)
}
