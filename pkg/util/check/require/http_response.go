package require

import (
	"net/http"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func ResponseMatchesFile(file string, successMsg, failureMsg string, otherFiles ...string) curl.HTTPResponseCheckFunc {
	return func(t test.TestHelper, resp *http.Response, responseBody []byte, duration time.Duration) {
		t.T().Helper()
		common.CheckResponseMatchesFile(t, resp, responseBody, file, successMsg, failureMsg, requireFailure, otherFiles...)
	}
}

func ResponseStatus(expectedStatus int) curl.HTTPResponseCheckFunc {
	return func(t test.TestHelper, resp *http.Response, responseBody []byte, duration time.Duration) {
		t.T().Helper()
		common.CheckResponseStatus(t, resp, expectedStatus, requireFailure)
	}
}

func ResponseContains(str string) curl.HTTPResponseCheckFunc {
	return func(t test.TestHelper, resp *http.Response, responseBody []byte, duration time.Duration) {
		t.T().Helper()
		common.CheckResponseContains(t, resp, responseBody, str, requireFailure)
	}
}

func DurationInRange(minDuration, maxDuration time.Duration) curl.HTTPResponseCheckFunc {
	return func(t test.TestHelper, resp *http.Response, responseBody []byte, duration time.Duration) {
		t.T().Helper()
		common.CheckDurationInRange(t, resp, duration, minDuration, maxDuration, requireFailure)
	}
}

func RequestFails(successMsg, failureMsg string) curl.HTTPResponseCheckFunc {
	return func(t test.TestHelper, resp *http.Response, responseBody []byte, duration time.Duration) {
		t.T().Helper()
		common.CheckRequestFails(t, resp, responseBody, successMsg, failureMsg, requireFailure)
	}
}
