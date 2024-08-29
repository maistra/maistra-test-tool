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

package assert

import (
	"net/http"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func ResponseMatchesFile(file string, successMsg, failureMsg string, otherFiles ...string) curl.HTTPResponseCheckFunc {
	return func(t test.TestHelper, resp *http.Response, responseBody []byte, responseErr error, duration time.Duration) {
		t.T().Helper()
		common.CheckResponseMatchesFile(t, resp, responseBody, file, successMsg, failureMsg, assertFailure, otherFiles...)
	}
}

func ResponseStatus(expectedStatus int) curl.HTTPResponseCheckFunc {
	return func(t test.TestHelper, resp *http.Response, responseBody []byte, responseErr error, duration time.Duration) {
		t.T().Helper()
		common.CheckResponseStatus(t, resp, responseBody, expectedStatus, assertFailure)
	}
}

func ResponseContains(str string) curl.HTTPResponseCheckFunc {
	return func(t test.TestHelper, resp *http.Response, responseBody []byte, responseErr error, duration time.Duration) {
		t.T().Helper()
		common.CheckResponseContains(t, resp, responseBody, str, assertFailure)
	}
}

func ResponseDoesNotContain(str string) curl.HTTPResponseCheckFunc {
	return func(t test.TestHelper, resp *http.Response, responseBody []byte, responseErr error, duration time.Duration) {
		t.T().Helper()
		common.CheckResponseDoesNotContain(t, resp, responseBody, str, assertFailure)
	}
}

func DurationInRange(minDuration, maxDuration time.Duration) curl.HTTPResponseCheckFunc {
	return func(t test.TestHelper, resp *http.Response, responseBody []byte, responseErr error, duration time.Duration) {
		t.T().Helper()
		common.CheckDurationInRange(t, resp, duration, minDuration, maxDuration, assertFailure)
	}
}

func RequestSucceeds(successMsg, failureMsg string) curl.HTTPResponseCheckFunc {
	return func(t test.TestHelper, resp *http.Response, responseBody []byte, responseErr error, duration time.Duration) {
		t.T().Helper()
		common.CheckRequestSucceeds(t, resp, responseBody, successMsg, failureMsg, assertFailure)
	}
}

func RequestFails(successMsg, failureMsg string) curl.HTTPResponseCheckFunc {
	return func(t test.TestHelper, resp *http.Response, responseBody []byte, responseErr error, duration time.Duration) {
		t.T().Helper()
		common.CheckRequestFails(t, resp, responseBody, successMsg, failureMsg, assertFailure)
	}
}

func RequestFailsWithErrorMessage(expectedErrorMessage, successMsg, failureMsg string) curl.HTTPResponseCheckFunc {
	return func(t test.TestHelper, resp *http.Response, responseBody []byte, responseErr error, duration time.Duration) {
		t.T().Helper()
		common.CheckRequestFailureMessagesAny(t, responseErr, []string{expectedErrorMessage}, successMsg, failureMsg, assertFailure)
	}
}

func RequestFailsWithAnyErrorMessages(expectedErrorMessages []string, successMsg string, failureMsg string) curl.HTTPResponseCheckFunc {
	return func(t test.TestHelper, resp *http.Response, responseBody []byte, responseErr error, duration time.Duration) {
		t.T().Helper()
		common.CheckRequestFailureMessagesAny(t, responseErr, expectedErrorMessages, successMsg, failureMsg, assertFailure)
	}
}
