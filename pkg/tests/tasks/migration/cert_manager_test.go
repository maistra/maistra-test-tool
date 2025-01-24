// Copyright 2025 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package migration

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/certmanageroperator"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/helm"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/template"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestCertManagerMigration(t *testing.T) {
	test.NewTest(t).MinVersion(version.SMCP_2_6).Groups(test.Migration).Run(func(t test.TestHelper) {
		t.Cleanup(func() {
			t.Log("Uninstalling cert-manager operator")
			certmanageroperator.Uninstall(t)
			oc.DeleteTestBoundNamespaces(t)
			// clean up bookinfo
			oc.DeleteFile(t, ns.Bookinfo, migrationGateway)
			app.Uninstall(t, app.Bookinfo(ns.Bookinfo))
		})
		ossm.BasicSetup(t)
		certmanageroperator.InstallIfNotExist(t)

		t.LogStep("Create intermediate certificate for Istio")
		oc.ApplyString(t, meshNamespace, istioCA)

		t.LogStep("Add jetstack repo to helm")
		helm.Repo("https://charts.jetstack.io").Add(t, "jetstack")

		smcp := ossm.DefaultClusterWideSMCP()
		smcp.Namespace = meshNamespace
		istio := ossm.DefaultIstio()
		istio.Template = istioWithCertManager
		istio.Namespace = meshNamespace

		istioCSRValues := map[string]any{
			"Namespace": meshNamespace,
			// The template doesn't apply slices as comma separated values
			// so we need to format the string ahead of time.
			"Revisions": strings.Join([]string{smcp.Name, istio.Name}, ","),
		}
		istioCSRTempl := template.Run(t, istioCSRTmpl, istioCSRValues)

		t.LogStepf("Install cert-manager-istio-csr with values:\n%s", istioCSRTempl)
		t.Cleanup(func() {
			t.Log("Uninstalling istio-csr helm chart")
			helm.Namespace(meshNamespace).Release("istio-csr").Uninstall(t)
		})
		helm.Namespace(meshNamespace).
			Chart("jetstack/cert-manager-istio-csr").
			Release("istio-csr").
			ValuesString(istioCSRTempl).
			Install(t)
		oc.WaitDeploymentRolloutComplete(t, meshNamespace, "cert-manager-istio-csr")

		t.LogStep("Deploy SMCP " + smcp.Version.String() + " and SMMR")
		oc.ApplyTemplate(t, meshNamespace, serviceMeshIstioCSRTmpl, smcp)
		oc.WaitSMCPReady(t, meshNamespace, smcp.Name)

		t.LogStep("Verify that istio-ca-root-cert created in Istio namespace")
		// Can take awhile for istio-csr to create all the configmaps.
		retry.UntilSuccessWithOptions(t, retry.Options().DelayBetweenAttempts(time.Second*4).MaxAttempts(80), func(t test.TestHelper) {
			oc.Get(t, meshNamespace, "ConfigMap", "istio-ca-root-cert")
		})

		t.LogStep(`Add injection label to "bookinfo" namespace`)
		oc.Label(t, "", "Namespace", ns.Bookinfo, "istio-injection=enabled")

		// Wait for SMMR to exist and include bookinfo
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.Get(t, smcp.Namespace, "servicemeshmemberroll", "default")
		})
		oc.WaitSMMRReady(t, smcp.Namespace)
		oc.DefaultOC.WaitFor(t, smcp.Namespace, "ServiceMeshMemberRoll", "default", `jsonpath='{.status.configuredMembers[?(@=="bookinfo")]}'`)

		t.LogStep("Install bookinfo and bookinfo gateway")
		app.InstallAndWaitReady(t, app.Bookinfo(ns.Bookinfo))
		oc.ApplyFile(t, ns.Bookinfo, migrationGateway)
		oc.DefaultOC.WaitDeploymentRolloutComplete(t, ns.Bookinfo, "bookinfo-gateway")
		oc.DefaultOC.WaitFor(t, ns.Bookinfo, "Route", "bookinfo-gateway", `jsonpath="{.status.ingress[].host}"`)
		hostname := oc.GetJson(t, ns.Bookinfo, "Routes", "bookinfo-gateway", "{.spec.host}")
		bookinfoGatewayURL := fmt.Sprintf("http://%s/productpage", hostname)

		t.Log("Enable strict mTLS for the whole mesh")
		oc.ApplyString(t, smcp.Namespace, enableMTLSPeerAuth)

		continuallyRequest(t, bookinfoGatewayURL)

		t.LogStep("Deploy Istio and IstioCNI")
		setupIstio(t, istio)

		t.LogStep("Ensure caBundle on webhooks match the CA in the istiod-tls secret")
		validatingWebhookName := fmt.Sprintf("istio-validator-%s-%s", istio.Name, meshNamespace)
		validatingWebhookCABundle := oc.GetJson(t, "", "validatingwebhookconfigurations", validatingWebhookName, "{.webhooks[0].clientConfig.caBundle}")
		istiodTLSCA := oc.GetJson(t, meshNamespace, "secrets", "istiod-tls", `{.data.ca\.crt}`)

		if validatingWebhookCABundle != istiodTLSCA {
			t.Fatalf("Validating Webhook '%s' caBundle is not equal to the istiod-tls CA.\nwebhookBundle: %s\nistiodTLSCA: %s\n", validatingWebhookName, validatingWebhookCABundle, istiodTLSCA)
		}
		t.Log("webhook caBundle matches istiod-tls ca cert")

		ensureResourceStable(t, validatingWebhookName, meshNamespace, "validatingwebhookconfigurations")

		t.LogStep("Migrate bookinfo to 3.0 controlplane")
		t.Log("Getting Istio active Rev name")
		ossm3RevName := oc.GetJson(t, "", "Istio", istio.Name, "{.status.activeRevisionName}")
		t.Log("Relabeling bookinfo namespace")
		oc.Label(t, "", "Namespace", ns.Bookinfo, maistraIgnoreLabel+" istio-injection- istio.io/rev="+ossm3RevName)
		// Wait for book info to be removed.
		retry.UntilSuccess(t, func(t test.TestHelper) {
			t.Log("Checking if \"bookinfo\" has been removed from default SMMR...")
			if namespaceInSMMR(t, ns.Bookinfo, "default", smcp.Namespace) {
				t.Error("bookinfo found in SMMR. Expected it to be removed.")
			}
		})
		t.Log("Bookinfo removed from SMMR. Restarting all workloads to inject new proxy that talk to new controlplane.")
		workloads := []workload{
			{Name: "productpage-v1", Labels: map[string]string{"app": "productpage", "version": "v1"}},
			{Name: "reviews-v1", Labels: map[string]string{"app": "reviews", "version": "v1"}},
			{Name: "reviews-v2", Labels: map[string]string{"app": "reviews", "version": "v2"}},
			{Name: "reviews-v3", Labels: map[string]string{"app": "reviews", "version": "v3"}},
			{Name: "ratings-v1", Labels: map[string]string{"app": "ratings", "version": "v1"}},
			{Name: "details-v1", Labels: map[string]string{"app": "details", "version": "v1"}},
			{Name: "bookinfo-gateway", Labels: map[string]string{"istio": "bookinfo-gateway"}},
		}
		oc.DefaultOC.RestartDeployments(t, ns.Bookinfo, workloadNames(workloads)...)
		// Waiting for the rollouts to complete ensures that old pods have been deleted.
		// If there are old pods lying around then the assertion below to get the pod annotations
		// will fail.
		oc.WaitDeploymentRolloutComplete(t, ns.Bookinfo, workloadNames(workloads)...)
		retry.UntilSuccess(t, func(t test.TestHelper) {
			if output := oc.DefaultOC.Invokef(t, `oc get pods -n %s -o jsonpath='{.items[?(@.metadata.deletionTimestamp!="")].metadata.name}'`, ns.Bookinfo); output != "" {
				t.Errorf("Pods still being deleted: %s", output)
			}
		})

		t.LogStep("Ensure all pods have migrated to 3.0 controlplane and curl requests succeed")
		for _, workload := range workloads {
			annotations := oc.GetPodAnnotations(t, pod.MatchingSelector(toSelector(workload.Labels), ns.Bookinfo))
			if actual := annotations["istio.io/rev"]; actual != ossm3RevName {
				t.Fatalf("Expected %s. Got: %s", ossm3RevName, actual)
			}
		}

		// One last request to ensure bookinfo still works.
		curl.Request(t, bookinfoGatewayURL, nil, assert.RequestSucceeds("productpage request succeeded", "productpage request failed"))
	})
}
