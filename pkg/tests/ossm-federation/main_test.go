package ossm_federation

import (
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/metallb"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestMain(m *testing.M) {
	test.NewSuite(m).
		Setup(func(t test.TestHelper) {
			if env.IsMetalLBInternalIPEnabled() && !env.IsRosa() {
				//Deploy MetalLB in both clusters only if MetalLB is enabled and the test is not running in ROSA
				metallb.InstallIfNotExist(t, env.GetKubeconfig(), env.GetKubeconfig2())
			}
		}).
		Run()
}
