package require

import (
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func requireFailure(t test.TestHelper, msg string, detailMsg string) {
	t.T().Helper()
	if detailMsg == "" {
		t.Fatalf(msg)
	} else {
		t.Fatalf("%s; %s", msg, detailMsg)
	}
}
