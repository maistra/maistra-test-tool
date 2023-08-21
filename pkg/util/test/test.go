package test

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

type TestGroup string

const (
	ARM          TestGroup = "arm"
	Full         TestGroup = "full"
	Smoke        TestGroup = "smoke"
	InterOp      TestGroup = "interop"
	Disconnected TestGroup = "disconnected"
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
	start := time.Now()
	th := &testHelper{t: t.t}
	defer func() {
		recoverPanic(t.t)
		t.t.Log()
		if th.Failed() {
			t.t.Logf("Test failed in %.2fs (excluding cleanup)", time.Now().Sub(start).Seconds())
			if env.IsMustGatherEnabled() {
				captureMustGather(t.t)
			}
		} else {
			t.t.Logf("Test completed in %.2fs (excluding cleanup)", time.Now().Sub(start).Seconds())
		}
	}()
	f(th)
}

func captureMustGather(t *testing.T) {
	image := env.GetMustGatherImage()
	dir := fmt.Sprintf("%s/failures-must-gather/%s-%s",
		env.GetOutputDir(),
		time.Now().Format("20060102150405"),
		strings.ReplaceAll(t.Name(), "/", "-"))

	t.Logf("Capturing cluster state using must-gather %s", image)
	cmd := exec.Command("sh", "-c", fmt.Sprintf(`rm -rf %s; mkdir -p %s; oc adm must-gather --dest-dir=%s --image=%s`, dir, dir, dir, image))
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Log(dir)
	} else {
		t.Logf("failed to create must-gather: %v", err)
	}
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
