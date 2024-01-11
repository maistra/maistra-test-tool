package operator

import (
	"fmt"
	"strings"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func GetCsvName(t test.TestHelper, operatorNamespace string, partialName string) string {
	output := shell.Execute(t, fmt.Sprintf(`oc get csv -n %s -o custom-columns="NAME:.metadata.name" |grep %s ||true`, operatorNamespace, partialName))
	return strings.TrimSpace(output)
}

func OperatorExists(t test.TestHelper, csvVersion string) bool {
	output := shell.Execute(t, fmt.Sprintf(`oc get csv -A -o custom-columns="NAME:.metadata.name,REPLACES:.spec.replaces" |grep %s ||true`, csvVersion))
	return strings.Contains(output, csvVersion)
}

func WaitForOperatorReady(t test.TestHelper, operatorNamespace string, operatorSelector string, csvName string) {
	t.Logf("Waiting for operator csv %s to succeed", csvName)
	// When the operator is installed, the CSV take some time to be created, need to wait until is created to validate the phase
	retry.UntilSuccessWithOptions(t, retry.Options().DelayBetweenAttempts(5*time.Second).MaxAttempts(70), func(t test.TestHelper) {
		if !OperatorExists(t, csvName) {
			t.Errorf("Operator csv %s is not yet installed", csvName)
		}
	})

	oc.WaitForPhase(t, operatorNamespace, "csv", csvName, "Succeeded")
	oc.WaitPodReadyWithOptions(t, retry.Options().MaxAttempts(70).DelayBetweenAttempts(5*time.Second), pod.MatchingSelector(operatorSelector, operatorNamespace))
}
