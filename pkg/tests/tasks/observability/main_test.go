package observability

import (
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	smcpName      = env.GetDefaultSMCPName()
	meshNamespace = env.GetDefaultMeshNamespace()
)

func TestMain(m *testing.M) {
	test.NewSuite(m).
		Setup(ossm.BasicSetup).
		Run()
}
