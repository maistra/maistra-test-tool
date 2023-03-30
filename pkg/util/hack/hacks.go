package hack

import (
	"github.com/sirupsen/logrus"

	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

// This is a temporary hack used in refactored tests, which disables all logs
// except the ones done via t.Log().
// We want to get to a point, where we only log via t.Log(). Until then, we
// want old tests to still use logrus, while the refactored tests use t.Log() and
// disable logrus.
func DisableLogrusForThisTest(t test.TestHelper) {
	originalLevel := log.Log.GetLevel()
	log.Log.SetLevel(logrus.ErrorLevel)
	t.Cleanup(func() {
		log.Log.SetLevel(originalLevel)
	})
}
