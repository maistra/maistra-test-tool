package extensions

import (
	_ "embed"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/test"

	"testing"
)

var (
	smcpName      = env.GetDefaultSMCPName()
	meshNamespace = env.GetDefaultMeshNamespace()
	threeScaleNs  = "3scale"

	//go:embed yaml/3scale-system.yaml
	threeScaleSystem string

	//go:embed yaml/3scale-system-service-entry.yaml
	threeScaleSystemSvcEntry string

	//go:embed yaml/3scale-backend.yaml
	threeScaleBackend string

	//go:embed yaml/3scale-backend-service-entry.yaml
	threeScaleBackendSvcEntry string

	//go:embed yaml/mesh.tmpl.yaml
	meshTmpl string

	//go:embed yaml/jwt-authn.tmpl.yaml
	jwtAuthnTmpl string

	//go:embed yaml/wasm-plugin.tmpl.yaml
	wasmPluginTmpl string
)

func TestMain(m *testing.M) {
	test.NewSuite(m).
		Setup(ossm.BasicSetup).
		Run()
}
