package test

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

func NewSetupTestHelper() TestHelper {
	return &setupTestHelper{}
}

var dummyT = &testing.T{}

type setupTestHelper struct {
}

var _ TestHelper = &setupTestHelper{}

func (t *setupTestHelper) Name() string {
	panic("not applicable")
}

func (t *setupTestHelper) Fail() {
	panic("Fail")
}

func (t *setupTestHelper) FailNow() {
	panic("FailNow")
}

func (t *setupTestHelper) Failed() bool {
	panic("not applicable")
}

func (t *setupTestHelper) Skip(args ...any) {
	panic("not applicable")
}

func (t *setupTestHelper) Skipf(format string, args ...any) {
	panic("not applicable")
}

func (t *setupTestHelper) SkipNow() {
	panic("not applicable")
}

func (t *setupTestHelper) Skipped() bool {
	panic("not applicable")
}

func (t *setupTestHelper) Log(args ...any) {
	log.Log.Info(args...)
}

func (t *setupTestHelper) Logf(format string, args ...any) {
	log.Log.Infof(format, args...)
}

func (t *setupTestHelper) Error(args ...any) {
	t.Log("ERROR: " + fmt.Sprint(args...))
	t.Fail()
}

func (t *setupTestHelper) Errorf(format string, args ...any) {
	t.Logf("ERROR: "+format, args...)
	t.Fail()
}

func (t *setupTestHelper) Fatal(args ...any) {
	t.Log("FATAL: " + fmt.Sprint(args...))
	t.FailNow()
}

func (t *setupTestHelper) Fatalf(format string, args ...any) {
	t.Logf("FATAL: "+format, args...)
	t.FailNow()
}

func (t *setupTestHelper) Cleanup(f func()) {
	panic("not applicable")
}

func (t *setupTestHelper) LogStep(str string) {
	panic("not applicable")
}

func (t *setupTestHelper) LogStepf(format string, args ...any) {
	panic("not applicable")
}

func (t *setupTestHelper) CurrentStep() int {
	panic("not applicable")
}

func (t *setupTestHelper) LogSuccess(str string) {
	panic("not applicable")
}

func (t *setupTestHelper) LogSuccessf(format string, args ...any) {
	panic("not applicable")
}

func (t *setupTestHelper) NewSubTest(name string) Test {
	panic("not applicable")
}

func (t *setupTestHelper) T() *testing.T {
	return dummyT
}

func (t *setupTestHelper) Parallel() {
	panic("not applicable")
}

func (t *setupTestHelper) WillRetry() bool {
	panic("not applicable")
}

func (t *setupTestHelper) indent() string {
	panic("not applicable")
}
