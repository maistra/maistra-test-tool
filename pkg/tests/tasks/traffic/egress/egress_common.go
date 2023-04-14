package egress

import (
	"github.com/maistra/maistra-test-tool/pkg/util"
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

func getCurlProxyParams() string {
	proxy, _ := util.GetProxy()
	curlParams := ""
	if proxy.HTTPProxy != "" {
		curlParams = "-x " + proxy.HTTPProxy
	}
	return curlParams
}
