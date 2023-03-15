package test

import (
	"testing"
)

type TestGroup string

const (
	ARM     TestGroup = "arm"
	Full    TestGroup = "full"
	Smoke   TestGroup = "smoke"
	InterOp TestGroup = "interop"
)

type Test interface {
	Run(f func(t TestHelper))
}

type TopLevelTest interface {
	Test
	Groups(groups ...TestGroup) TopLevelTest
	LegacyID(ids ...string) TopLevelTest
}

func NewTest(t *testing.T) TopLevelTest {
	return &topLevelTest{t: t}
}

var _ Test = &topLevelTest{}

type topLevelTest struct {
	t         *testing.T
	groups    []TestGroup
	legacyIDs []string
}

func (t *topLevelTest) Groups(groups ...TestGroup) TopLevelTest {
	t.groups = groups
	return t
}

func (t *topLevelTest) LegacyID(id ...string) TopLevelTest {
	t.legacyIDs = id
	return t
}

func (t *topLevelTest) Run(f func(t TestHelper)) {
	defer recoverPanic(t.t)
	f(&testHelper{t: t.t})
}
