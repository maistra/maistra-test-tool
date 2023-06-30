package certmanageroperator

import (
	_ "embed"
	"fmt"
	"strings"

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
	certmanagerVersion    = "cert-manager-operator.v1.11.1"
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
	output := shell.Execute(t, fmt.Sprintf("oc get csv %s -n cert-manager-operator -o name||true", certmanagerVersion))
	return !strings.Contains(output, "NotFound")
}

func installOperator(t test.TestHelper) {
	t.LogStep("Create namespace for cert-manager-operator")
	oc.CreateNamespace(t, certManagerOperatorNs)

	t.LogStep("Install cert-manager-operator")
	oc.ApplyTemplate(t, certManagerOperatorNs, certManagerOperator, map[string]string{"Version": certmanagerVersion})
}

func waitOperatorSucceded(t test.TestHelper, certManagerOperatorNs string) {
	t.Log("Waiting for cert-manager-operator to succeed")
	oc.WaitFor(t, certManagerOperatorNs, "csv", certmanagerVersion, "jsonpath='{.status.phase}'=Succeeded")

	retry.UntilSuccess(t, func(t test.TestHelper) {
		//This is a hack to wait for the pods to be ready because the pods app=cert-manager take a long time to be ready
		//With this hack we will avoid flaky tests and we do not increase the timeout for the entire oc.WaitPodReady
		oc.WaitPodReady(t, pod.MatchingSelector("name=cert-manager-operator", certManagerOperatorNs))
		oc.WaitPodReady(t, pod.MatchingSelector("app=cert-manager", certManagerNs))
	})
}
