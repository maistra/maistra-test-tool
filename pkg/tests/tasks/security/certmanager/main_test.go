package certmanager

import (
	_ "embed"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	smcpName      = env.GetDefaultSMCPName()
	meshNamespace = env.GetDefaultMeshNamespace()

	//go:embed yaml/istio-csr/istio-ca.yaml
	istioCA string

	//go:embed yaml/istio-csr/mesh.yaml
	serviceMeshIstioCsrTmpl string

	//go:embed yaml/istio-csr/istio-csr.yaml
	istioCsrTmpl string

	//go:embed yaml/cacerts/mesh.yaml
	serviceMeshCacertsTmpl string

	//go:embed yaml/cacerts/cacerts.yaml
	cacerts string
)

func TestMain(m *testing.M) {
	test.NewSuite(m).
		Setup(setupCertManagerOperator).
		Run()
}

func setupCertManagerOperator(t test.TestHelper) {
	ossm.BasicSetup(t)
}
