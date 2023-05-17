package certmanager

import (
	_ "embed"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	smcpName              = env.GetDefaultSMCPName()
	meshNamespace         = env.GetDefaultMeshNamespace()
	certManagerOperatorNs = "cert-manager-operator"
	certManagerNs         = "cert-manager"

	//go:embed yaml/cert-manager-operator.yaml
	certManagerOperator string

	//go:embed yaml/istio-csr.yaml
	istioCsrTmpl string

	//go:embed yaml/mesh.yaml
	meshTmpl string

	//go:embed yaml/root-ca.yaml
	rootCA string

	//go:embed yaml/istio-ca.yaml
	istioCA string
)

func TestMain(m *testing.M) {
	test.NewSuite(m).
		Setup(setupCertManagerOperator).
		Run()
}

func setupCertManagerOperator(t test.TestHelper) {
	t.Cleanup(func() {
		oc.DeleteFromString(t, certManagerNs, rootCA)
		oc.DeleteFromString(t, certManagerOperatorNs, certManagerOperator)
		oc.DeleteNamespace(t, certManagerOperatorNs)
		oc.DeleteNamespace(t, certManagerNs)
	})
	ossm.BasicSetup(t)

	t.LogStep("Create namespace for cert-manager-operator")
	oc.CreateNamespace(t, certManagerOperatorNs)

	t.LogStep("Install cert-manager-operator")
	oc.ApplyString(t, certManagerOperatorNs, certManagerOperator)
	oc.WaitPodReady(t, pod.MatchingSelector("name=cert-manager-operator", certManagerOperatorNs))
	oc.WaitPodReady(t, pod.MatchingSelector("app=cert-manager", certManagerNs))

	t.LogStep("Create root ca")
	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.ApplyString(t, certManagerNs, rootCA)
	})
}
