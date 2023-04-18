package test

import (
	"fmt"
	"testing"
	"time"
)

const (
	SuccessPrefix = "SUCCESS: "
	FailurePrefix = "FAILURE: "
)

func NewTestContext(t *testing.T) TestHelper {
	ctx := &testHelper{
		t: t,
	}
	return ctx
}

type TestHelper interface {
	Name() string
	Cleanup(f func())
	Fail()
	FailNow()
	Failed() bool
	Skip(args ...any)
	Skipf(format string, args ...any)
	SkipNow()
	Skipped() bool
	Error(args ...any)
	Errorf(format string, args ...any)
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Log(args ...any)
	Logf(format string, args ...any)

	NewSubTest(name string) Test

	LogStep(str string)
	LogStepf(format string, args ...any)
	CurrentStep() int

	LogSuccess(str string)
	LogSuccessf(format string, args ...any)

	T() *testing.T

	Parallel()

	WillRetry() bool
}

type testHelper struct {
	t           *testing.T
	currentStep int
}

var _ TestHelper = &testHelper{}

func (t *testHelper) Name() string {
	return t.t.Name()
}

func (t *testHelper) Fail() {
	t.t.Fail()
}

func (t *testHelper) FailNow() {
	t.t.FailNow()
}

func (t *testHelper) Failed() bool {
	return t.t.Failed()
}

func (t *testHelper) Skip(args ...any) {
	t.t.Skip(args...)
}

func (t *testHelper) Skipf(format string, args ...any) {
	t.t.Skipf(format, args...)
}

func (t *testHelper) SkipNow() {
	t.t.SkipNow()
}

func (t *testHelper) Skipped() bool {
	return t.t.Skipped()
}

func (t *testHelper) Log(args ...any) {
	t.t.Helper()
	t.t.Log(t.indent() + fmt.Sprint(args...))
}

func (t *testHelper) Logf(format string, args ...any) {
	t.t.Helper()
	t.t.Logf(t.indent()+format, args...)
}

func (t *testHelper) Error(args ...any) {
	t.t.Helper()
	t.Log(FailurePrefix + fmt.Sprint(args...))
	t.Fail()
}

func (t *testHelper) Errorf(format string, args ...any) {
	t.t.Helper()
	t.Logf(FailurePrefix+format, args...)
	t.Fail()
}

func (t *testHelper) Fatal(args ...any) {
	t.t.Helper()
	t.Log("FATAL: " + fmt.Sprint(args...))
	t.FailNow()
}

func (t *testHelper) Fatalf(format string, args ...any) {
	t.t.Helper()
	t.Logf("FATAL: "+format, args...)
	t.FailNow()
}

func (t *testHelper) Cleanup(f func()) {
	t.t.Helper()
	t.t.Cleanup(func() {
		t.T().Helper()
		start := time.Now()
		t.T().Log()
		t.T().Log("Performing cleanup")
		f()
		t.T().Logf("Cleanup completed in %.2fs", time.Now().Sub(start).Seconds())
	})
}

func (t *testHelper) LogStep(str string) {
	t.t.Helper()
	t.currentStep++
	t.Log("")
	t.t.Logf("STEP %d: %s", t.currentStep, str)
}

func (t *testHelper) LogStepf(format string, args ...any) {
	t.t.Helper()
	t.LogStep(fmt.Sprintf(format, args...))
}

func (t *testHelper) CurrentStep() int {
	return t.currentStep
}

func (t *testHelper) LogSuccess(str string) {
	t.t.Helper()
	t.Log(SuccessPrefix + str)
}

func (t *testHelper) LogSuccessf(format string, args ...any) {
	t.t.Helper()
	t.LogSuccess(fmt.Sprintf(format, args...))
}

func (t *testHelper) NewSubTest(name string) Test {
	return subTest{
		t:    t.t,
		name: name,
	}
}

func (t *testHelper) T() *testing.T {
	return t.t
}

func (t *testHelper) Parallel() {
	t.t.Parallel()
}

func (t *testHelper) WillRetry() bool {
	return false
}

func (t *testHelper) indent() string {
	if t.currentStep > 0 {
		return "   "
	}
	return ""
}
