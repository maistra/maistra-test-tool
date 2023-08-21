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
