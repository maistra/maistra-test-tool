package retry

import (
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func UntilSuccess(t test.TestHelper, f func(t test.TestHelper)) {
	t.T().Helper()
	UntilSuccessWithOptions(t, defaultOptions, f)
}

func UntilSuccessWithOptions(t test.TestHelper, options RetryOptions, f func(t test.TestHelper)) {
	t.T().Helper()
	start := time.Now()
	for i := 0; i < options.maxAttempts; i++ {
		lastAttempt := i == options.maxAttempts-1

		var attemptHelper test.TestHelper
		if lastAttempt {
			attemptHelper = t
			f(attemptHelper)
		} else {
			retryTestHelper := test.NewRetryTestHelper(t.T(), t.CurrentStep(), i, options.maxAttempts)
			attemptHelper = retryTestHelper
			retryTestHelper.Attempt(f)
		}

		if attemptHelper.Failed() {
			if lastAttempt {
				if options.logAttempts {
					t.Logf("Last attempt (%d/%d) failed.", i+1, options.maxAttempts)
				}
			} else {
				if options.logAttempts {
					if options.delayBetweenAttempts == defaultOptions.delayBetweenAttempts {
						t.Logf("--- Attempt %d/%d failed. Retrying...", i+1, options.maxAttempts)
					} else {
						t.Logf("--- Attempt %d/%d failed. Retrying in %v...", i+1, options.maxAttempts, options.delayBetweenAttempts)
					}
				}
				time.Sleep(options.delayBetweenAttempts)
			}
		} else {
			if i > 0 && options.logAttempts {
				// there was at least one failed attempt, so let's log the current attempt as successful so that
				// the user isn't left wondering
				t.Logf("--- Attempt %d/%d successful; total time: %.2fs", i+1, options.maxAttempts, time.Now().Sub(start).Seconds())
			}
			if options.maxAttempts > 1 {
				percentage := i * 100 / options.maxAttempts
				if percentage >= 90 {
					t.Log("WARNING: This test is is almost certainly flaky since it required more than 90% of the maximum retry count to succeed. Consider increasing the maximum retry count to prevent flakiness.")
				} else if percentage >= 75 {
					t.Log("WARNING: This test may be flaky since it required more than 75% of the maximum retry count to succeed. Consider increasing the maximum retry count to prevent flakiness.")
				}
			}
			break
		}
	}
}
