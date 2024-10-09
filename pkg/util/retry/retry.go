// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package retry

import (
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
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
			attemptHelper = attemptInternal(t, f, i, options.maxAttempts)
		}

		if attemptHelper.Failed() {
			if lastAttempt {
				if options.logAttempts && env.IsLogFailedRetryAttempts() {
					t.Logf("Last attempt (%d/%d) failed.", i+1, options.maxAttempts)
				}
				t.T().FailNow()
			} else {
				if options.logAttempts && env.IsLogFailedRetryAttempts() {
					if options.delayBetweenAttempts == defaultOptions.delayBetweenAttempts {
						t.Logf("--- Attempt %d/%d failed. Retrying...", i+1, options.maxAttempts)
					} else {
						t.Logf("--- Attempt %d/%d failed. Retrying in %v...", i+1, options.maxAttempts, options.delayBetweenAttempts)
					}
				}
				time.Sleep(options.delayBetweenAttempts)
			}
		} else {
			if env.IsLogFailedRetryAttempts() {
				if i > 0 && options.logAttempts {
					// there was at least one failed attempt, so let's log the current attempt as successful so that
					// the user isn't left wondering
					t.Logf("--- Attempt %d/%d successful; total time: %.2fs", i+1, options.maxAttempts, time.Now().Sub(start).Seconds())
				}
			} else {
				// this attempt was successful, so we must flush the log buffer to display the SUCCESS messages
				if retryTestHelper, ok := attemptHelper.(*test.RetryTestHelper); ok {
					retryTestHelper.FlushLogBuffer()
				}
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

// Attempt runs the given function, captures any errors thrown by the function, and
// returns a RetryTestHelper, which you can use to:
// - check if the attempt failed by invoking retryTestHelper.Failed()
// - print everything that the function logged by invoking retryTestHelper.FlushLogBuffer()
func Attempt(t test.TestHelper, f func(t test.TestHelper)) *test.RetryTestHelper {
	t.T().Helper()
	return attemptInternal(t, f, 0, 1)
}

func attemptInternal(t test.TestHelper, f func(t test.TestHelper), currentAttempt, maxAttempts int) *test.RetryTestHelper {
	t.T().Helper()
	retryTestHelper := test.NewRetryTestHelper(t.T(), t.CurrentStep(), currentAttempt, maxAttempts)
	retryTestHelper.Attempt(f)
	return retryTestHelper
}
