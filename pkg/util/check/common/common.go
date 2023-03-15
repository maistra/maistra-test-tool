package common

import (
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func logSuccess(t test.TestHelper, msg string) {
	t.T().Helper()
	t.Log("SUCCESS:", msg)
}
