// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
)

func NewSetupTestHelper() TestHelper {
	return &setupTestHelper{
		log: newSetupLogger(),
		t:   &testing.T{},
	}
}

type setupTestHelper struct {
	log *logrus.Logger
	t   *testing.T

	cleanup func()
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
	t.t.Helper()
	t.t.Skip(args...)
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
	t.log.Info(args...)
}

func (t *setupTestHelper) Logf(format string, args ...any) {
	t.log.Infof(format, args...)
}

func (t *setupTestHelper) Error(args ...any) {
	t.Log(FailurePrefix + fmt.Sprint(args...))
	t.Fail()
}

func (t *setupTestHelper) Errorf(format string, args ...any) {
	t.Logf(FailurePrefix+format, args...)
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
	t.cleanup = f
}

func (t *setupTestHelper) LogStep(str string) {
	t.Logf("SETUP: %s", str)
}

func (t *setupTestHelper) LogStepf(format string, args ...any) {
	panic("not applicable")
}

func (t *setupTestHelper) CurrentStep() int {
	return 0
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
	return t.t
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
