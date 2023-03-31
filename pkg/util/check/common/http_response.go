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
	requireNonNilResponse(t, resp)

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
			matchedFile := findMatchingFile(body, otherFiles)
			if matchedFile == "" {
				detailMsg = fmt.Sprintf("expected the response to match file %q, but it didn't match that or any other file", file)
				if !t.WillRetry() {
					detailMsg += "\ndiff between the expected and actual response:\n" + err.Error()
				}
			} else {
				detailMsg = fmt.Sprintf("expected the response to match file %q, but it matched %q", file, matchedFile)
			}
		} else {
			detailMsg = fmt.Sprintf("expected the response to match file %q, but it didn't", file)
			if !t.WillRetry() {
				detailMsg += "\ndiff between the expected and actual response:\n" + err.Error()
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

func CheckResponseStatus(t test.TestHelper, resp *http.Response, expectedStatus int, failure FailureFunc) {
	t.T().Helper()
	requireNonNilResponse(t, resp)
	if resp.StatusCode != expectedStatus {
		failure(t, fmt.Sprintf("expected status code %d but got %s", expectedStatus, resp.Status), "")
	}
}

func CheckResponseContains(t test.TestHelper, resp *http.Response, str string, failure FailureFunc) {
	t.T().Helper()
	requireNonNilResponse(t, resp)

	defer util.CloseResponseBody(resp)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	body := string(bodyBytes)
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

func CheckDurationInRange(t test.TestHelper, resp *http.Response, duration, minDuration, maxDuration time.Duration, failure FailureFunc) {
	t.T().Helper()
	requireNonNilResponse(t, resp)

	if minDuration <= duration && duration <= maxDuration {
		logSuccess(t, fmt.Sprintf("request completed in %v, which is within the expected range %v - %v", duration.Truncate(time.Millisecond), minDuration, maxDuration))
	} else {
		failure(t, fmt.Sprintf("expected request duration to be between %v and %v, but was %v", minDuration, maxDuration, duration.Truncate(time.Millisecond)), "")
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
			t.Fatalf("Failed to read response body: %v", err)
		}

		detailMsg := fmt.Sprintf("expected request to fail, but it succeeded with the following status: %s", resp.Status)
		if !t.WillRetry() {
			detailMsg += "\nfull response:\n" + string(bodyBytes)
		}
		failure(t, failureMsg, detailMsg)
	}
}

func requireNonNilResponse(t test.TestHelper, resp *http.Response) {
	t.T().Helper()
	if resp == nil {
		t.Fatal("response is nil; the HTTP request must have failed")
	}
}
