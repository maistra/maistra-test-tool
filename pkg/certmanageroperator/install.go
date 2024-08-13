package certmanageroperator

import (
	_ "embed"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/operator"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/cert-manager-operator.yaml
	certManagerSubscriptionYaml string

	//go:embed yaml/root-ca.yaml
	rootCA string

	certManagerOperatorNs = "cert-manager-operator"
	certManagerNs         = "cert-manager"

	certManagerCSVName          = "cert-manager-operator"
	certManagerOperatorSelector = "name=cert-manager-operator"
)

func InstallIfNotExist(t test.TestHelper) {
	if oc.ResourceByLabelExists(t, certManagerOperatorNs, "pod", "name=cert-manager-operator") {
		t.Log("cert-manager-operator is already installed")
	} else {
		t.Log("cert-manager-operator is not installed, starting installation")
		install(t)
	}
}

func install(t test.TestHelper) {
	t.LogStep("Create namespace for cert-manager-operator")
	oc.CreateNamespace(t, certManagerOperatorNs)

	t.LogStep("Install cert-manager-operator")
	operator.CreateOperatorViaOlm(t, certManagerOperatorNs, certManagerCSVName, certManagerSubscriptionYaml, certManagerOperatorSelector, nil)

	t.LogStep("Wait for cert manager control plane")
	oc.WaitPodReadyWithOptions(t, retry.Options().MaxAttempts(70).DelayBetweenAttempts(5*time.Second), pod.MatchingSelector("app=cert-manager", certManagerNs))
	oc.WaitPodReadyWithOptions(t, retry.Options().MaxAttempts(70).DelayBetweenAttempts(5*time.Second), pod.MatchingSelector("app=cainjector", certManagerNs))
	oc.WaitPodReadyWithOptions(t, retry.Options().MaxAttempts(70).DelayBetweenAttempts(5*time.Second), pod.MatchingSelector("app=webhook", certManagerNs))

	t.LogStep("Wait for cert-manager-webhook service available")
	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.Get(t,
			certManagerNs,
			"service",
			"cert-manager-webhook",
			assert.OutputDoesNotContain("NotFound",
				"Service \"cert-manager-webhook\" found",
				"Service \"cert-manager-webhook\" not found"))
	})

	t.LogStep("Create root ca")
	oc.ApplyString(t, certManagerNs, rootCA)
}

func Uninstall(t test.TestHelper) {
	oc.DeleteFromString(t, certManagerNs, rootCA)
	operator.DeleteOperatorViaOlm(t, certManagerOperatorNs, certManagerCSVName, certManagerSubscriptionYaml)
	oc.DeleteNamespace(t, certManagerOperatorNs)
	oc.DeleteNamespace(t, certManagerNs)
}
