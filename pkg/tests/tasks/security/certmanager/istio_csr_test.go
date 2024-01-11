package certmanager

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/certmanageroperator"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/helm"
	"github.com/maistra/maistra-test-tool/pkg/util/istio"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/template"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestIstioCsr(t *testing.T) {
	test.NewTest(t).Id("T38").Groups(test.Full, test.ARM).Run(func(t test.TestHelper) {
		//Validate OCP version, this test setup can't be executed in OCP versions less than 4.12
		//More information in: https://57747--docspreview.netlify.app/openshift-enterprise/latest/service_mesh/v2x/ossm-security.html#ossm-cert-manager-integration-istio_ossm-security
		smcpVer := env.GetSMCPVersion()
		if smcpVer.LessThan(version.SMCP_2_4) {
			t.Skip("istio-csr is not supported in SMCP older than v2.4")
		}
		ocpVersion := version.ParseVersion(oc.GetOCPVersion(t))
		if ocpVersion.LessThan(version.OCP_4_12) {
			t.Skip("istio-csr is not supported in OCP older than v4.12")
		}
		if env.GetArch() == "z" || env.GetArch() == "p" {
			t.Skip("istio-csr is not supported for IBM Z&P")
		}

		meshValues := map[string]string{
			"Name":    smcpName,
			"MeshNs":  meshNamespace,
			"Member":  ns.Foo,
			"Version": smcpVer.String(),
		}
		istioCsrValues := map[string]string{
			"MeshNs":   meshNamespace,
			"Revision": smcpName,
		}

		t.Cleanup(func() {
			helm.Namespace(meshNamespace).Release("istio-csr").Uninstall(t)
			oc.DeleteFromTemplate(t, meshNamespace, serviceMeshIstioCsrTmpl, meshValues)
			oc.DeleteFromString(t, meshNamespace, istioCA)
			oc.DeleteSecret(t, meshNamespace, "istiod-tls")
			oc.DeleteSecret(t, meshNamespace, "istio-ca")
			oc.RecreateNamespace(t, ns.Foo)
			certmanageroperator.Uninstall(t)
		})

		certmanageroperator.InstallIfNotExist(t)

		t.LogStep("Uninstall existing SMCP")
		oc.RecreateNamespace(t, meshNamespace)

		t.LogStep("Create intermediate certificate for Istio")
		oc.ApplyString(t, meshNamespace, istioCA)

		t.LogStep("Add jetstack repo to helm")
		helm.Repo("https://charts.jetstack.io").Add(t, "jetstack")

		t.LogStep("Install cert-manager-istio-csr")
		helm.Namespace(meshNamespace).
			Chart("jetstack/cert-manager-istio-csr").
			Release("istio-csr").
			Version("v0.6.0").
			ValuesString(template.Run(t, istioCsrTmpl, istioCsrValues)).
			Install(t)
		oc.WaitDeploymentRolloutComplete(t, meshNamespace, "cert-manager-istio-csr")

		t.LogStep("Deploy SMCP " + smcpVer.String() + " and SMMR")
		oc.ApplyTemplate(t, meshNamespace, serviceMeshIstioCsrTmpl, meshValues)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Verify that istio-ca-root-cert created in Istio and member namespaces")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.LogsFromPods(t, meshNamespace, "app=cert-manager-istio-csr",
				assertIstioCARootCertCreatedOrUpdated(meshNamespace),
				assertIstioCARootCertCreatedOrUpdated(ns.Foo))
		})

		t.LogStep("Verify that istio-ca-root-cert not created in non-member namespaces")
		oc.LogsFromPods(t, meshNamespace, "app=cert-manager-istio-csr",
			assert.OutputDoesNotContain(
				fmt.Sprintf(`"msg"="creating configmap with root CA data" "configmap"="istio-ca-root-cert" "namespace"="%s"`, ns.Bar),
				fmt.Sprintf("istio-ca-root-cert not created in %s", ns.Bar),
				fmt.Sprintf("istio-ca-root-cert created in %s", ns.Bar)))

		t.LogStep("Deploy httpbin and sleep")
		app.InstallAndWaitReady(t, app.Httpbin(ns.Foo), app.Sleep(ns.Foo))

		t.LogStep("Check if httpbin returns 200 OK ")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.Exec(t,
				pod.MatchingSelector("app=sleep", ns.Foo),
				"sleep",
				`curl http://httpbin:8000/ip -s -o /dev/null -w "%{http_code}"`,
				assert.OutputContains(
					"200",
					"Got expected 200 OK from httpbin",
					"Expected 200 OK from httpbin, but got a different HTTP code"))
		})

		t.LogStep("Check mTLS traffic from ingress gateway to httpbin")
		oc.ApplyFile(t, ns.Foo, "https://raw.githubusercontent.com/maistra/istio/maistra-2.5/samples/httpbin/httpbin-gateway.yaml")
		httpbinURL := fmt.Sprintf("http://%s/headers", istio.GetIngressGatewayHost(t, meshNamespace))
		retry.UntilSuccess(t, func(t test.TestHelper) {
			curl.Request(t, httpbinURL, nil, assert.ResponseStatus(http.StatusOK))
		})
	})
}

func assertIstioCARootCertCreatedOrUpdated(ns string) common.CheckFunc {
	return assert.OutputContainsAny(
		[]string{
			fmt.Sprintf(`"msg"="creating configmap with root CA data" "configmap"="istio-ca-root-cert" "namespace"="%s"`, ns),
			fmt.Sprintf(`"msg"="updating ConfigMap data" "configmap"="istio-ca-root-cert" "namespace"="%s"`, ns),
		},
		fmt.Sprintf("istio-ca-root-cert created or updated in %s", ns),
		fmt.Sprintf("istio-ca-root-cert neither created nor updated in %s", ns))
}
