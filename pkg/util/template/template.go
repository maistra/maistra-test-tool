package template

import (
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func Run(t test.TestHelper, yaml string, vars interface{}) string {
	t.T().Helper()
	return util.RunTemplateWithTestHelper(t, yaml, vars)
}

type SMCP struct {
	Name      string `default:"basic"`
	Namespace string `default:"istio-system"`
	Rosa      bool   `default:"false"`
}
