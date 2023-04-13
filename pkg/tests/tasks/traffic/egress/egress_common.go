package egress

import (
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func execInSleepPod(t test.TestHelper, ns string, command string, checks ...common.CheckFunc) {
	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.Exec(t, pod.MatchingSelector("app=sleep", ns), "sleep",
			command, checks...)
	})
}
