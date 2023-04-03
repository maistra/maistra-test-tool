package assert

import (
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func assertFailure(t test.TestHelper, msg string, detailMsg string) {
	t.T().Helper()
	if detailMsg == "" {
		t.Error(msg)
	} else {
		t.Errorf("%s\n%s", msg, detailMsg)
	}
}
