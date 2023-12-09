package certmanageroperator

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/cert-manager-operator.yaml
	certManagerOperator string

	//go:embed yaml/root-ca.yaml
	rootCA string

	certManagerOperatorNs = "cert-manager-operator"
	certManagerNs         = "cert-manager"
	certmanagerVersion    = "cert-manager-operator.v1.12.1"
)

func InstallIfNotExist(t test.TestHelper) {
	if certManagerOperatorExists(t) {
		t.Log("cert-manager-operator is already installed")
	} else {
		t.Log("cert-manager-operator is not installed, starting installation")
		install(t)
	}
}

func install(t test.TestHelper) {
	installOperator(t)
	waitOperatorSucceded(t, certManagerOperatorNs)

	t.LogStep("Create root ca")
	oc.ApplyString(t, certManagerNs, rootCA)

}

func Uninstall(t test.TestHelper) {
	oc.DeleteFromString(t, certManagerNs, rootCA)
	oc.DeleteFromTemplate(t, certManagerOperatorNs, certManagerOperator, map[string]string{"Version": certmanagerVersion})
	oc.DeleteNamespace(t, certManagerOperatorNs)
	oc.DeleteNamespace(t, certManagerNs)
}

func certManagerOperatorExists(t test.TestHelper) bool {
	output := shell.Execute(t, fmt.Sprintf(`oc get csv -A -o custom-columns="NAME:.metadata.name,REPLACES:.spec.replaces" |grep %s ||true`, certmanagerVersion))
	return strings.Contains(output, certmanagerVersion)
}

func installOperator(t test.TestHelper) {
	t.LogStep("Create namespace for cert-manager-operator")
	oc.CreateNamespace(t, certManagerOperatorNs)

	t.LogStep("Install cert-manager-operator")
	oc.ApplyTemplate(t, certManagerOperatorNs, certManagerOperator, map[string]string{"Version": certmanagerVersion})
}

func waitOperatorSucceded(t test.TestHelper, certManagerOperatorNs string) {
	t.Log("Waiting for cert-manager-operator to succeed")
	// When the operator is installed, the CSV take some time to be created, need to wait until is created to validate the phase
	retry.UntilSuccessWithOptions(t, retry.Options().DelayBetweenAttempts(5*time.Second).MaxAttempts(70), func(t test.TestHelper) {
		if !certManagerOperatorExists(t) {
			t.Error("cert-manager-operator is not yet installed")
		}
	})

	oc.WaitForPhase(t, certManagerOperatorNs, "csv", certmanagerVersion, "Succeeded")
	oc.WaitPodReadyWithOptions(t, retry.Options().MaxAttempts(70).DelayBetweenAttempts(5*time.Second), pod.MatchingSelector("name=cert-manager-operator", certManagerOperatorNs))
	oc.WaitPodReadyWithOptions(t, retry.Options().MaxAttempts(70).DelayBetweenAttempts(5*time.Second), pod.MatchingSelector("app=cert-manager", certManagerNs))
}
