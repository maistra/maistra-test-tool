package certificate

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/helm"
	"github.com/maistra/maistra-test-tool/pkg/util/istio"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestCertManager(t *testing.T) {
	test.NewTest(t).Id("T38").Groups(test.Full, test.ARM, test.InterOp).Run(func(t test.TestHelper) {
		smcpVer := env.GetSMCPVersion()
		if smcpVer.LessThan(version.SMCP_2_3) {
			t.Skip("cert-manager-istio-csr is not supported in SMCP older than v2.3.2")
		}

		certManagerOperatorNs := "cert-manager-operator"
		certManagerNs := "cert-manager"
		t.Cleanup(func() {
			helm.Namespace(meshNamespace).Release("istio-csr").Uninstall(t)
			oc.DeleteFromString(t, meshNamespace, istioCA)
			oc.DeleteFromString(t, certManagerNs, rootCA)
			oc.DeleteFromString(t, certManagerOperatorNs, certManagerOperator)
			oc.DeleteSecret(t, meshNamespace, "istiod-tls")
			oc.DeleteSecret(t, meshNamespace, "istio-ca")
			oc.DeleteNamespace(t, certManagerNs)
			oc.DeleteNamespace(t, certManagerOperatorNs)
			oc.RecreateNamespace(t, ns.Foo)
		})

		t.LogStep("Uninstall existing SMCP")
		oc.RecreateNamespace(t, meshNamespace)

		t.LogStep("Create namespace for cert-manager-operator")
		oc.CreateNamespace(t, certManagerOperatorNs)

		t.LogStep("Install cert-manager-operator")
		oc.ApplyString(t, certManagerOperatorNs, certManagerOperator)
		oc.WaitPodReady(t, pod.MatchingSelector("name=cert-manager-operator", certManagerOperatorNs))
		oc.WaitPodReady(t, pod.MatchingSelector("app=cert-manager", certManagerNs))

		t.LogStep("Create certificates")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.ApplyString(t, certManagerNs, rootCA)
			oc.ApplyString(t, meshNamespace, istioCA)
		})

		t.LogStep("Add jetstack repo to helm")
		helm.Repo("https://charts.jetstack.io").Add(t, "jetstack")

		t.LogStep("Install cert-manager-istio-csr")
		helm.Namespace(meshNamespace).
			Chart("jetstack/cert-manager-istio-csr").
			Release("istio-csr").
			Version("v0.6.0").
			ValuesString(istioCsrValues(meshNamespace, smcpName)).
			Install(t)
		oc.WaitPodsReady(t, meshNamespace, "app=cert-manager-istio-csr")

		t.LogStep("Deploy SMCP " + smcpVer.String())
		oc.ApplyString(t, meshNamespace, createSMCPWithCertManager(smcpName, meshNamespace, ns.Foo, smcpVer.String()))
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Verify that istio-ca-root-cert created in proper namespaces")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.LogsFromPods(t, meshNamespace, "app=cert-manager-istio-csr",
				assert.OutputContains(
					fmt.Sprintf(`"msg"="creating configmap with root CA data" "configmap"="istio-ca-root-cert" "namespace"="%s"`, meshNamespace),
					fmt.Sprintf("istio-ca-root-cert created in %s", meshNamespace),
					fmt.Sprintf("istio-ca-root-cert not created in %s", meshNamespace)))
			oc.LogsFromPods(t, meshNamespace, "app=cert-manager-istio-csr",
				assert.OutputContains(
					fmt.Sprintf(`"msg"="creating configmap with root CA data" "configmap"="istio-ca-root-cert" "namespace"="%s"`, ns.Foo),
					fmt.Sprintf("istio-ca-root-cert created in %s", ns.Foo),
					fmt.Sprintf("istio-ca-root-cert not created in %s", ns.Foo)))
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
		oc.ApplyFile(t, ns.Foo, "https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/httpbin/httpbin-gateway.yaml")
		httpbinURL := fmt.Sprintf("http://%s/headers", istio.GetIngressGatewayHost(t, meshNamespace))
		retry.UntilSuccess(t, func(t test.TestHelper) {
			curl.Request(t, httpbinURL, nil, assert.ResponseStatus(http.StatusOK))
		})
	})
}

func createSMCPWithCertManager(smcpName, smcpNamespace, memberNs, version string) string {
	return fmt.Sprintf(`
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: %s
spec:
  addons:
    grafana:
      enabled: false
    kiali:
      enabled: false
    prometheus:
      enabled: false
  gateways:
    egress:
      enabled: false
    openshiftRoute:
      enabled: false
  security:
    certificateAuthority:
      cert-manager:
        address: cert-manager-istio-csr.%s.svc:443
      type: cert-manager
    dataPlane:
      mtls: true
    identity:
      type: ThirdParty
  tracing:
    type: None
  version: %s
---
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:
  - %s
`, smcpName, smcpNamespace, version, memberNs)
}

func istioCsrValues(meshNamespace, smcpName string) string {
	return fmt.Sprintf(`
replicaCount: 2

image:
  repository: quay.io/jetstack/cert-manager-istio-csr
  tag: v0.6.0
  pullSecretName: ""

app:
  certmanager:
    namespace: %[1]s
    issuer:
      group: cert-manager.io
      kind: Issuer
      name: istio-ca

  controller:
    configmapNamespaceSelector: "maistra.io/member-of=%[1]s"
    leaderElectionNamespace: %[1]s

  istio:
    namespace: %[1]s
    revisions: ["%[2]s"]

  server:
    maxCertificateDuration: 5m

  tls:
    certificateDNSNames:
    # This DNS name must be set in the SMCP spec.security.certificateAuthority.cert-manager.address
    - cert-manager-istio-csr.%[1]s.svc
`, meshNamespace, smcpName)
}

const (
	certManagerOperator = `
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: cert-manager-operator
spec:
  targetNamespaces:
  - cert-manager-operator
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: openshift-cert-manager-operator
  labels:
    operators.coreos.com/openshift-cert-manager-operator.cert-manager-operator: ""
spec:
  channel: stable-v1
  installPlanApproval: Automatic
  name: openshift-cert-manager-operator
  source: redhat-operators
  sourceNamespace: openshift-marketplace
  startingCSV: cert-manager-operator.v1.10.2
`

	rootCA = `
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: selfsigned-root-issuer
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: selfsigned-ca
spec:
  isCA: true
  duration: 21600h # 900d
  secretName: root-ca
  commonName: root-ca.my-company.net
  subject:
    organizations:
    - my-company.net
  issuerRef:
    name: selfsigned-root-issuer
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: root-ca
spec:
  ca:
    secretName: root-ca
`
	istioCA = `
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: istio-ca
spec:
  isCA: true
  duration: 21600h
  secretName: istio-ca
  commonName: istio-ca.my-company.net
  subject:
    organizations:
    - my-company.net
  issuerRef:
    name: root-ca
    kind: ClusterIssuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: istio-ca
spec:
  ca:
    secretName: istio-ca
`
)
