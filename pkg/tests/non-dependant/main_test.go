package non_dependant

import (
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestMain(m *testing.M) {
	test.NewSuite(m).
		Setup(SetupNamespacesAndControlPlane).
		Run()
}
