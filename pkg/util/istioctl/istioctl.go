package istioctl

import (
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func CheckClusters(t test.TestHelper, podLocator oc.PodLocatorFunc, checks ...common.CheckFunc) {
	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.Exec(t, podLocator, "istio-proxy", "curl localhost:15000/clusters", checks...)
	})
}
