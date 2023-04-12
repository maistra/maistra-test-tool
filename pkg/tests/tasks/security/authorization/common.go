package authorization

import (
	"fmt"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func httpbinRequest(method string, path string, headers ...string) string {
	headerArgs := ""
	for _, header := range headers {
		headerArgs += fmt.Sprintf(` -H "%s"`, header)
	}
	return fmt.Sprintf(`curl "http://httpbin:8000%s" -X %s%s -sS -o /dev/null -w "%%%%{http_code}\n"`, path, method, headerArgs)
}

func assertHttpbinRequestSucceeds(t test.TestHelper, ns string, curlCommand string) {
	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.Exec(t,
			pod.MatchingSelector("app=sleep", ns),
			"sleep",
			curlCommand,
			assert.OutputContains(
				"200",
				"Got expected 200 OK from httpbin",
				"Expected 200 OK from httpbin, but got a different HTTP code"))
	})
}

func assertRequestAccepted(t test.TestHelper, ns string, curlCommand string) {
	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.Exec(t,
			pod.MatchingSelector("app=sleep", ns),
			"sleep",
			curlCommand,
			assert.OutputContains(
				"200",
				"Got the expected 200 OK response for request from httpbin",
				"Expected the AuthorizationPolicy to accept request (expected HTTP status 200), but got a different HTTP code"))
	})
}

func assertRequestDenied(t test.TestHelper, ns string, curlCommand string, expectedStatusCode string) {
	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.Exec(t,
			pod.MatchingSelector("app=sleep", ns),
			"sleep",
			curlCommand,
			assert.OutputContains(
				expectedStatusCode,
				fmt.Sprintf("Got the expected %s response code", expectedStatusCode),
				fmt.Sprintf("Expected the AuthorizationPolicy to reject request (expected HTTP status %s), but got a different HTTP code", expectedStatusCode)))
	})
}
