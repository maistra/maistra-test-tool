package authorizaton

import (
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestMain(m *testing.M) {
	test.NewSuite(m).
		Setup(ossm.SetupNamespacesAndControlPlane).
		Run()
}
