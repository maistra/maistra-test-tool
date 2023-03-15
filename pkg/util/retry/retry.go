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

		if i > 0 {
			t.Log()
			t.Logf("Attempt %d/%d:", i+1, options.maxAttempts)
		}

		var attemptHelper test.TestHelper
		if lastAttempt {
			attemptHelper = t
			f(attemptHelper)
		} else {
			retryTestHelper := test.NewRetryTestHelper(t.T(), i, options.maxAttempts)
			attemptHelper = retryTestHelper
			retryTestHelper.Attempt(f)
		}

		if attemptHelper.Failed() {
			if lastAttempt {
				t.Logf("Last attempt (%d/%d) failed.", i+1, options.maxAttempts)
			} else {
				t.Logf("Attempt %d/%d failed. Retrying...", i+1, options.maxAttempts)
				time.Sleep(options.delayBetweenAttempts)
			}
		} else {
			if i > 0 {
				// there was at least one failed attempt, so let's log the current attempt as successful so that
				// the user isn't left wondering
				t.Logf("Attempt %d/%d successful; total time: %v", i+1, options.maxAttempts, time.Now().Sub(start))
			}
			break
		}
	}
}
