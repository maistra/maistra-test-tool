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
