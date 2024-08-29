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
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
)

type subTest struct {
	t    *testing.T
	name string
}

var _ Test = subTest{}

func (t subTest) Run(f func(t TestHelper)) {
	t.t.Helper()
	t.t.Run(t.name, func(t *testing.T) {
		t.Helper()
		start := time.Now()
		th := NewTestHelper(t)
		defer func() {
			recoverPanic(t)
			t.Log()
			if th.Failed() {
				t.Logf("Subtest failed in %.2fs (excluding cleanup)", time.Now().Sub(start).Seconds())
				if env.IsMustGatherEnabled() {
					captureMustGather(t)
				}
			} else {
				t.Logf("Subtest completed in %.2fs (excluding cleanup)", time.Now().Sub(start).Seconds())
			}
		}()
		f(th)
	})
}

// recover from panic if one occurred. This allows cleanup to be executed after panic.
func recoverPanic(t *testing.T) {
	t.Helper()
	if err := recover(); err != nil {
		t.Errorf("Test panic: %v", err)
	}
}
