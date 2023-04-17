package certificate

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/helm"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestCertManager(t *testing.T) {
	test.NewTest(t).Id("T38").Groups(test.Full, test.ARM, test.InterOp).Run(func(t test.TestHelper) {
		ns := "foo"
		certManagerNs := "cert-manager"

		t.Cleanup(func() {
			helm.Namespace("istio-system").Release("istio-csr").Uninstall(t)
			helm.Namespace("cert-manager").Release("cert-manager").Uninstall(t)
			oc.DeleteNamespace(t, certManagerNs)
			oc.RecreateNamespace(t, ns)
		})

		t.LogStep("Uninstall the SMCP")
		oc.RecreateNamespace(t, "istio-system")
		oc.CreateNamespace(t, certManagerNs)

		t.LogStep("Add jetstack repo to helm")
		helm.Repo("https://charts.jetstack.io").Add(t, "jetstack")

		t.LogStep("Install cert-manager")
		helm.Namespace(certManagerNs).
			Chart("jetstack/cert-manager").
			Release("cert-manager").
			Version("v1.11.0").
			Set("installCRDs=true").
			Install(t)
		oc.WaitPodsReady(t, certManagerNs, "app=cert-manager")

		t.LogStep("Provision root certificate")
		oc.ApplyString(t, certManagerNs, rootCA)

		t.LogStep("Provision Istio certificate")
		oc.ApplyString(t, meshNamespace, istioCA)

		t.LogStep("Install cert-manager-istio-csr")
		helm.Namespace("istio-system").
			Chart("jetstack/cert-manager-istio-csr").
			Release("istio-csr").
			ValuesStdIn(istioCsrValues(meshNamespace, smcpName)).
			Install(t)
		oc.WaitPodsReady(t, "istio-system", "app=cert-manager-istio-csr")

		t.LogStep("Deploy the cert-manager in SMCP")
		oc.ApplyString(t, "istio-system", createSMCPWithCertManager(smcpName, meshNamespace, ns))
		oc.WaitSMCPReady(t, "istio-system", smcpName)

		t.LogStep("Install httpbin and sleep")
		app.InstallAndWaitReady(t, app.Httpbin(ns), app.Sleep(ns))

		t.LogStep("Check if httpbin returns 200 OK ")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.Exec(t,
				pod.MatchingSelector("app=sleep", ns),
				"sleep",
				`curl http://httpbin:8000/ip -s -o /dev/null -w "%{http_code}"`,
				assert.OutputContains(
					"200",
					"Got expected 200 OK from httpbin",
					"Expected 200 OK from httpbin, but got a different HTTP code"))
		})

	})
}

func createSMCPWithCertManager(smcpName, smcpNamespace, memberNs string) string {
	return fmt.Sprintf(`apiVersion: maistra.io/v2
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
  version: v2.3
---
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:
  - %s
`, smcpName, smcpNamespace, memberNs)
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
