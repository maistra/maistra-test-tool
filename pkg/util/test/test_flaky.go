package test

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
)

const flakinessCheckUsingSubTests = false

var stopFlakinessCheckOnFirstSuccess = env.Getenv("STOP_FLAKINESS_CHECK_ON_FIRST_SUCCESS", "false") == "true"
var maxFlakinessCheckRuns = env.GetenvAsInt("MAX_FLAKINESS_CHECK_RUNS", 10)

func NewTestWithFlakinessDetection(t *testing.T) Test {
	return flakinessDetectorTest{t: t}
}

var _ Test = flakinessDetectorTest{}

type flakinessDetectorTest struct {
	t *testing.T
}

// Run runs the test. If the test run is successful, it doesn't run it again.
// If the first test run fails, the test is run 9 more times to see whether
// it will succeed on one of the other test runs. If it does, it's a flaky
// test. If it doesn't, it's just a failed test.
func (t flakinessDetectorTest) Run(f func(t TestHelper)) {
	t.t.Helper()
	defer recoverPanic(t.t)

	passCount := 0
	failCount := 0
	for attempt := 0; attempt < maxFlakinessCheckRuns; attempt++ {
		var passed bool
		if flakinessCheckUsingSubTests {
			if attempt == 0 {
				passed = runTest(t.t, f, attempt)
			} else {
				t.t.Run(fmt.Sprintf("investigate-flakiness-%d", attempt), func(t *testing.T) {
					passed = runTest(t, f, attempt)
					if !passed {
						// t.Fail()
						t.SkipNow()
					}
				})
			}
		} else {
			passed = runTest(t.t, f, attempt)
		}

		if passed {
			passCount++
			if attempt == 0 {
				// test passed on the first run, so we don't re-run it
				break
			} else {
				// test passed after failing initially; run it again to see how many times it fails
				if stopFlakinessCheckOnFirstSuccess {
					break
				}
			}
		} else {
			failCount++
			if attempt == 0 {
				t.t.Log("TEST FAILED: Retesting to detect possible flakiness")
			}
		}
	}
	if passCount > 0 && failCount > 0 {
		var additionalInfo string
		if stopFlakinessCheckOnFirstSuccess {
			additionalInfo = fmt.Sprintf("passed after %d attempts", passCount+failCount)
		} else {
			additionalInfo = fmt.Sprintf("passed %d/%d times", passCount, passCount+failCount)
		}
		t.t.Log()
		// t.t.Logf("FLAKY TEST DETECTED: %s (%s)", t.t.Name(), additionalInfo)
		t.t.Logf("WARNING: %s is flaky: %s", t.t.Name(), additionalInfo)
	} else if failCount == maxFlakinessCheckRuns {
		t.t.Log()
		t.t.Fatalf("TEST FAILED %d TIMES", failCount)
	}
}

func runTest(t *testing.T, f func(t TestHelper), attempt int) bool {
	t.Helper()
	if attempt > 0 {
		// t.Log("")
		// if !flakinessCheckUsingSubTests {
		// 	fmt.Println()
		// }
		// fmt.Printf("    === RERUN #%d: %s\n", attempt, t.Name())
		t.Log()
		t.Logf("=== RERUN #%d: %s\n", attempt, t.Name())
	}
	th := NewFlakyTestHelper(t)
	f(th)
	passed := !th.Failed()
	if attempt > 0 {
		if passed {
			// t.Logf("--- PASS: %s (re-run %d)", t.Name(), attempt)
			// fmt.Printf("    --- PASS: %s (rerun #%d)\n", t.Name(), attempt)
			t.Logf("--- PASS: %s (rerun #%d)\n", t.Name(), attempt)
		} else {
			// t.Logf("--- FAIL: %s (re-run %d)", t.Name(), attempt)
			// fmt.Printf("    --- FAIL: %s (rerun #%d)\n", t.Name(), attempt)
			t.Logf("--- FAIL: %s (rerun #%d)\n", t.Name(), attempt)
		}
	}
	return passed
}
