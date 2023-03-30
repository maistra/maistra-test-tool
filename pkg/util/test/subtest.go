package test

import (
	"testing"
)

type subTest struct {
	t    *testing.T
	name string
}

var _ Test = subTest{}

func (t subTest) Run(f func(t TestHelper)) {
	t.t.Run(t.name, func(t *testing.T) {
		defer recoverPanic(t)
		ctx := NewTestContext(t)
		f(ctx)
	})
}

// recover from panic if one occurred. This allows cleanup to be executed after panic.
func recoverPanic(t *testing.T) {
	t.Helper()
	if err := recover(); err != nil {
		t.Errorf("Test panic: %v", err)
	}
}
