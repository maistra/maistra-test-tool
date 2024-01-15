package certmanageroperator

import (
	_ "embed"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/operator"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
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
	if operator.OperatorExists(t, certmanagerVersion) {
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

func installOperator(t test.TestHelper) {
	t.LogStep("Create namespace for cert-manager-operator")
	oc.CreateNamespace(t, certManagerOperatorNs)

	t.LogStep("Install cert-manager-operator")
	oc.ApplyTemplate(t, certManagerOperatorNs, certManagerOperator, map[string]string{"Version": certmanagerVersion})
}

func waitOperatorSucceded(t test.TestHelper, certManagerOperatorNs string) {
	operator.WaitForOperatorReady(t, certManagerOperatorNs, "name=cert-manager-operator", certmanagerVersion)
	oc.WaitPodReadyWithOptions(t, retry.Options().MaxAttempts(70).DelayBetweenAttempts(5*time.Second), pod.MatchingSelector("app=cert-manager", certManagerNs))
}
