package test

import (
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
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
	MinVersion(v version.Version) TopLevelTest
	MaxVersion(v version.Version) TopLevelTest
	Id(id string) TopLevelTest
}

func NewTest(t *testing.T) TopLevelTest {
	return &topLevelTest{t: t}
}

var _ Test = &topLevelTest{}

type topLevelTest struct {
	t          *testing.T
	id         string
	groups     []TestGroup
	minVersion *version.Version
	maxVersion *version.Version
}

func (t *topLevelTest) Groups(groups ...TestGroup) TopLevelTest {
	t.groups = groups
	return t
}

func (t *topLevelTest) MinVersion(v version.Version) TopLevelTest {
	t.minVersion = &v
	return t
}

func (t *topLevelTest) MaxVersion(v version.Version) TopLevelTest {
	t.maxVersion = &v
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
	f(th)
	t.t.Log()
	t.t.Logf("Test completed in %.2fs (excluding cleanup)", time.Now().Sub(start).Seconds())
}

func (t *topLevelTest) skipIfNecessary() {
	testGroup := TestGroup(env.GetTestGroup())
	if env.GetArch() == "arm" {
		testGroup = "arm"
	}

	if !t.isPartOfGroup(testGroup) {
		t.t.Skipf("This test is being skipped because it is not part of the %q test group", testGroup)
	}

	smcpVersion := env.GetSMCPVersion()
	if t.minVersion != nil && smcpVersion.LessThan(*t.minVersion) {
		t.t.Skipf("This test is being skipped because it doesn't support the current SMCP version %s (min version is %s)", smcpVersion, t.minVersion)
	}
	if t.maxVersion != nil && smcpVersion.GreaterThan(*t.maxVersion) {
		t.t.Skipf("This test is being skipped because it doesn't support the current SMCP version %s (max version is %s)", smcpVersion, t.maxVersion)
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
