package common

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type FailureFunc func(t test.TestHelper, msg string, detailedMsg string)

func CheckResponseMatchesFile(t test.TestHelper, resp *http.Response, file, successMsg, failureMsg string, failure FailureFunc, otherFiles ...string) {
	t.T().Helper()
	requireNonNilResponse(t, resp, failure)

	defer util.CloseResponseBody(resp)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	err = util.CompareHTTPResponse(body, file)
	if err == nil {
		if successMsg == "" {
			successMsg = fmt.Sprintf("response matches file %s", file)
		}
		logSuccess(t, successMsg)
	} else {
		var detailMsg string
		if len(otherFiles) > 0 {
			matchedFile := ""
			for _, otherFile := range otherFiles {
				if matchesFile(body, otherFile) {
					matchedFile = otherFile
					break
				}
			}
			if matchedFile == "" {
				if t.WillRetry() {
					detailMsg = fmt.Sprintf("expected response to match file %q, but it didn't; it also didn't match any of the other files", file)
				} else {
					detailMsg = fmt.Sprintf("expected response to match file %q, but it didn't; it also didn't match any of the other files; diff:\n%s", file, err)
				}
			} else {
				detailMsg = fmt.Sprintf("expected response to match file %q, but it matched %q", file, matchedFile)
			}
		} else {
			if t.WillRetry() {
				detailMsg = fmt.Sprintf("expected response to match file %q, but it didn't", file)
			} else {
				detailMsg = fmt.Sprintf("expected response to match file %q, but it differs:\n%s", file, err)
			}
		}
		failure(t, failureMsg, detailMsg)
	}
}

func matchesFile(body []byte, file string) bool {
	err := util.CompareHTTPResponse(body, file)
	return err == nil
}

func CheckResponseStatus(t test.TestHelper, resp *http.Response, expectedStatus int, failure FailureFunc) {
	t.T().Helper()
	requireNonNilResponse(t, resp, failure)
	if resp.StatusCode != expectedStatus {
		failure(t, fmt.Sprintf("expected status code %d but got %s", expectedStatus, resp.Status), "")
	}
}

func CheckResponseContains(t test.TestHelper, resp *http.Response, str string, failure FailureFunc) {
	t.T().Helper()
	requireNonNilResponse(t, resp, failure)

	defer util.CloseResponseBody(resp)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		failure(t, fmt.Sprintf("Failed to read response body: %v", err), "")
	}
	body := string(bodyBytes)
	if strings.Contains(body, str) {
		logSuccess(t, fmt.Sprintf("found %q in response", str))
	} else {
		failure(t, fmt.Sprintf("expected response to contain %q, but it didn't; the response was: %v", str, body), "")
	}
}

func CheckDurationInRange(t test.TestHelper, resp *http.Response, duration, minDuration, maxDuration time.Duration, failure FailureFunc) {
	t.T().Helper()
	requireNonNilResponse(t, resp, failure)

	if minDuration <= duration && duration <= maxDuration {
		logSuccess(t, fmt.Sprintf("request duration was %v (within range %v - %v)", duration, minDuration, maxDuration))
	} else {
		failure(t, fmt.Sprintf("expected request duration to be between %v and %v, but was %v", minDuration, maxDuration, duration), "")
	}
}

func CheckRequestFails(t test.TestHelper, resp *http.Response, successMsg, failureMsg string, failure FailureFunc) {
	t.T().Helper()
	if resp == nil {
		if successMsg != "" {
			logSuccess(t, successMsg)
		}
	} else {
		defer util.CloseResponseBody(resp)
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			failure(t, fmt.Sprintf("Failed to read response body: %v", err), "")
		}

		detailMsg := fmt.Sprintf("expected request to fail, but it didn't; status: %s, body: %v", resp.Status, string(bodyBytes))
		failure(t, failureMsg, detailMsg)
	}
}

func requireNonNilResponse(t test.TestHelper, resp *http.Response, failure FailureFunc) {
	t.T().Helper()
	if resp == nil {
		t.Fatalf("response is nil; the HTTP request must have failed")
	}
}
