package certmanager

import (
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	smcpName              = env.GetDefaultSMCPName()
	meshNamespace         = env.GetDefaultMeshNamespace()
	certManagerOperatorNs = "cert-manager-operator"
	certManagerNs         = "cert-manager"
)

func TestMain(m *testing.M) {
	test.NewSuite(m).
		Setup(ossm.BasicSetup).
		Run()
}
