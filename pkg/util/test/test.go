package test

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
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
	Id(id string) TopLevelTest
	NotRefactoredYet()
}

func NewTest(t *testing.T) TopLevelTest {
	return &topLevelTest{t: t}
}

var _ Test = &topLevelTest{}

type topLevelTest struct {
	t      *testing.T
	id     string
	groups []TestGroup
}

func (t *topLevelTest) Groups(groups ...TestGroup) TopLevelTest {
	t.groups = groups
	return t
}

func (t *topLevelTest) Id(id string) TopLevelTest {
	t.id = id
	return t
}

func (t *topLevelTest) Run(f func(t TestHelper)) {
	t.t.Helper()
	t.skipIfNecessary()
	defer recoverPanic(t.t)
	start := time.Now()
	th := &testHelper{t: t.t}
	disableLogrusForThisTest(th)
	f(th)
	t.t.Log()
	t.t.Logf("Test completed in %.2fs (excluding cleanup)", time.Now().Sub(start).Seconds())
}

func (t *topLevelTest) NotRefactoredYet() {
	t.skipIfNecessary()
}

func (t *topLevelTest) skipIfNecessary() {
	testGroup := TestGroup(env.Getenv("TEST_GROUP", string(Full)))
	if env.Getenv("SAMPLEARCH", "x86") == "arm" {
		testGroup = "arm"
	}

	if !t.isPartOfGroup(testGroup) {
		t.t.Skipf("This test is being skipped because it is not part of the %q test group", testGroup)
	}
}

func (t *topLevelTest) isPartOfGroup(group TestGroup) bool {
	for _, g := range t.groups {
		if g == group {
			return true
		}
	}
	return false
}

// This is a temporary hack used in refactored tests, which disables all logs
// except the ones done via t.Log().
// We want to get to a point, where we only log via t.Log(). Until then, we
// want old tests to still use logrus, while the refactored tests use t.Log() and
// disable logrus.
func disableLogrusForThisTest(t TestHelper) {
	originalLevel := log.Log.GetLevel()
	log.Log.SetLevel(logrus.ErrorLevel)
	t.T().Cleanup(func() {
		log.Log.SetLevel(originalLevel)
	})
}
