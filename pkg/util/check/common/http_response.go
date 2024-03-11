package common

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type FailureFunc func(t test.TestHelper, msg string, detailedMsg string)

func CheckResponseMatchesFile(t test.TestHelper, resp *http.Response, responseBody []byte, file, successMsg, failureMsg string, failure FailureFunc, otherFiles ...string) {
	t.T().Helper()
	requireNonNilResponse(t, resp)

	err := util.CompareHTTPResponse(responseBody, file)
	if err == nil {
		if successMsg == "" {
			successMsg = fmt.Sprintf("response matches file %s", file)
		}
		logSuccess(t, successMsg)
	} else {
		var detailMsg string
		if len(otherFiles) > 0 {
			matchedFile := findMatchingFile(responseBody, otherFiles)
			if matchedFile == "" {
				detailMsg = fmt.Sprintf("expected the response to match file %q, but it didn't match that or any other file", file)
				if !t.WillRetry() {
					detailMsg += "\ndiff between the expected (-) and actual response (+):\n" + err.Error()
				}
			} else {
				detailMsg = fmt.Sprintf("expected the response to match file %q, but it matched %q", file, matchedFile)
			}
		} else {
			detailMsg = fmt.Sprintf("expected the response to match file %q, but it didn't", file)
			if !t.WillRetry() {
				detailMsg += "\ndiff between the expected (-) and actual response (+):\n" + err.Error()
			}
		}
		failure(t, failureMsg, detailMsg)
	}
}

func findMatchingFile(body []byte, otherFiles []string) string {
	for _, file := range otherFiles {
		if matchesFile(body, file) {
			return file
		}
	}
	return ""
}

func matchesFile(body []byte, file string) bool {
	err := util.CompareHTTPResponse(body, file)
	return err == nil
}

func CheckResponseStatus(t test.TestHelper, resp *http.Response, responseBody []byte, expectedStatus int, failure FailureFunc) {
	t.T().Helper()
	requireNonNilResponse(t, resp)
	if resp.StatusCode == expectedStatus {
		logSuccess(t, fmt.Sprintf("received expected status code %d", expectedStatus))
	} else {
		if t.WillRetry() {
			failure(t, fmt.Sprintf("expected status code %d but got %s", expectedStatus, resp.Status), "")
		} else {
			failure(t, fmt.Sprintf("expected status code %d but got %s and response: %s", expectedStatus, resp.Status, string(responseBody)), "")
		}
	}
}

func CheckResponseContains(t test.TestHelper, resp *http.Response, responseBody []byte, str string, failure FailureFunc) {
	t.T().Helper()
	requireNonNilResponse(t, resp)

	body := string(responseBody)
	if strings.Contains(body, str) {
		logSuccess(t, fmt.Sprintf("string '%s' found in response", str))
	} else {
		detailMsg := fmt.Sprintf("expected to find the string '%s' in the response, but it wasn't found", str)
		if !t.WillRetry() {
			detailMsg += "\nfull response:\n" + body
		}
		failure(t, detailMsg, "")
	}
}

func CheckResponseDoesNotContain(t test.TestHelper, resp *http.Response, responseBody []byte, str string, failure FailureFunc) {
	t.T().Helper()
	requireNonNilResponse(t, resp)

	body := string(responseBody)
	if strings.Contains(body, str) {
		detailMsg := fmt.Sprintf("expected the string '%s' to be absent from the response, but it was present", str)
		if !t.WillRetry() {
			detailMsg += "\nfull response:\n" + body
		}
		failure(t, detailMsg, "")
	} else {
		logSuccess(t, fmt.Sprintf("string '%s' not found in response", str))
	}
}

func CheckDurationInRange(t test.TestHelper, resp *http.Response, duration, minDuration, maxDuration time.Duration, failure FailureFunc) {
	t.T().Helper()
	requireNonNilResponse(t, resp)

	if minDuration <= duration && duration <= maxDuration {
		logSuccess(t, fmt.Sprintf("request completed in %v, which is within the expected range %v - %v", duration.Truncate(time.Millisecond), minDuration, maxDuration))
	} else {
		failure(t, fmt.Sprintf("expected request duration to be between %v and %v, but was %v", minDuration, maxDuration, duration.Truncate(time.Millisecond)), "")
	}
}

func CheckRequestSucceeds(t test.TestHelper, resp *http.Response, responseBody []byte, successMsg, failureMsg string, failure FailureFunc) {
	if resp == nil {
		failure(t, failureMsg, "expected request to succeed, but it failed")
	} else if successMsg != "" {
		logSuccess(t, successMsg)
	}
}

func CheckRequestFails(t test.TestHelper, resp *http.Response, responseBody []byte, successMsg, failureMsg string, failure FailureFunc) {
	t.T().Helper()
	if resp == nil {
		if successMsg != "" {
			logSuccess(t, successMsg)
		}
	} else {
		detailMsg := fmt.Sprintf("expected request to fail, but it succeeded with the following status: %s", resp.Status)
		if !t.WillRetry() {
			detailMsg += "\nfull response:\n" + string(responseBody)
		}
		failure(t, failureMsg, detailMsg)
	}
}

func CheckRequestFailureMessagesAny(t test.TestHelper, requestError error, expectedErrorMessages []string, successMsg, failureMsg string, failure FailureFunc) {
	t.T().Helper()
	if requestError == nil {
		failure(t, "expected request error, but it is nil", "")
	}
	for _, str := range expectedErrorMessages {
		if strings.Contains(requestError.Error(), str) {
			if successMsg != "" {
				logSuccess(t, successMsg)
			}
			return
		}
	}
	// none of the expected strings were found
	detailMsg := fmt.Sprintf("\nexpected any of error messages:'%s'\nactual error message:'%s'", expectedErrorMessages, requestError.Error())
	failure(t, failureMsg, detailMsg)
}

func requireNonNilResponse(t test.TestHelper, resp *http.Response) {
	t.T().Helper()
	if resp == nil {
		t.Fatal("response is nil; the HTTP request must have failed")
	}
}
