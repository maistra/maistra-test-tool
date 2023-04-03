package test

import (
	"testing"
	"time"
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
		defer recoverPanic(t)
		ctx := NewTestContext(t)
		start := time.Now()
		f(ctx)
		t.Logf("Subtest completed in %.2fs (excluding cleanup)", time.Now().Sub(start).Seconds())
	})
}

// recover from panic if one occurred. This allows cleanup to be executed after panic.
func recoverPanic(t *testing.T) {
	t.Helper()
	if err := recover(); err != nil {
		t.Errorf("Test panic: %v", err)
	}
}
