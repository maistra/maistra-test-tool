package certmanager

import (
	_ "embed"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

var (
	smcpName              = env.GetDefaultSMCPName()
	meshNamespace         = env.GetDefaultMeshNamespace()
	certManagerOperatorNs = "cert-manager-operator"
	certManagerNs         = "cert-manager"

	//go:embed yaml/cert-manager-operator.yaml
	certManagerOperator string

	//go:embed yaml/root-ca.yaml
	rootCA string

	//go:embed yaml/istio-csr/istio-csr.yaml
	istioCsrTmpl string

	//go:embed yaml/istio-csr/mesh.yaml
	serviceMeshIstioCsrTmpl string

	//go:embed yaml/istio-csr/istio-ca.yaml
	istioCA string

	//go:embed yaml/cacerts/mesh.yaml
	serviceMeshCacertsTmpl string

	//go:embed yaml/cacerts/cacerts.yaml
	cacerts string
)

func TestMain(m *testing.M) {
	test.NewSuite(m).
		Setup(setupCertManagerOperator).
		Run()
}

func setupCertManagerOperator(t test.TestHelper) {
	//Validate OCP version, this test setup can't be executed in OCP versions less than 4.12
	//More information in: https://57747--docspreview.netlify.app/openshift-enterprise/latest/service_mesh/v2x/ossm-security.html#ossm-cert-manager-integration-istio_ossm-security
	ocpVersion := version.ParseVersion(oc.GetOCPVersion(t))
	//Verify if the OCP version is less than 4.12
	if ocpVersion.LessThan(version.OCP_4_12) {
		t.Log("WARNING: This test setup can't be executed in OCP versions less than 4.12")
		return
	}
	smcpVer := env.GetSMCPVersion()
	if smcpVer.LessThan(version.SMCP_2_4) {
		t.Log("WARNING: This test setup can't be executed in SMCP versions less than 2.4")
		return
	}

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
	oc.ApplyString(t, certManagerNs, rootCA)
}
