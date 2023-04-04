package template

import (
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func Run(t test.TestHelper, yaml string, vars interface{}) string {
	t.T().Helper()
	return util.RunTemplate(yaml, vars)
}
