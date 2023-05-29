package egress

import (
	"fmt"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func execInSleepPod(t test.TestHelper, ns string, command string, checks ...common.CheckFunc) {
	t.T().Helper()
	retry.UntilSuccess(t, func(t test.TestHelper) {
		t.T().Helper()
		oc.Exec(t, pod.MatchingSelector("app=sleep", ns), "sleep", command, checks...)
	})
}

func assertRequestSuccess(t test.TestHelper, client app.App, url string) {
	execInSleepPod(t, client.Namespace(), buildGetRequestCmd(url),
		assert.OutputContains("200",
			fmt.Sprintf("Got expected 200 OK from %s", url),
			fmt.Sprintf("Expect 200 OK from %s, but got a different HTTP code", url)))
}

func assertRequestFailure(t test.TestHelper, client app.App, url string) {
	execInSleepPod(t, client.Namespace(), buildGetRequestCmd(url),
		assert.OutputContains(curlFailedMessage,
			"Got a failure message as expected",
			"Expect request to failed, but got a response"))
}

func assertInsecureRequestSuccess(t test.TestHelper, client app.App, url string) {
	url = fmt.Sprintf(`curl -sSL --insecure -o /dev/null -w "%%{http_code}" %s 2>/dev/null || echo %s`, url, curlFailedMessage)
	execInSleepPod(t, client.Namespace(), url,
		assert.OutputContains("200",
			fmt.Sprintf("Got expected 200 OK from %s", url),
			fmt.Sprintf("Expect 200 OK from %s, but got a different HTTP code", url)))
}

func buildGetRequestCmd(location string) string {
	return fmt.Sprintf(`curl -sSL -o /dev/null -w "%%{http_code}" %s 2>/dev/null || echo %s`, location, curlFailedMessage)
}

const (
	curlFailedMessage = "CURL_FAILED"
)
