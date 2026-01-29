// Copyright 2026 Red Hat, Inc.
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
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/cert"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestCustomCAMigration(t *testing.T) {
	test.NewTest(t).MinVersion(version.SMCP_2_6).Groups(test.Migration).Run(func(t test.TestHelper) {
		t.Cleanup(func() {
			oc.DeleteTestBoundNamespaces(t)
			oc.DeleteFile(t, ns.Bookinfo, migrationGateway)
			app.Uninstall(t, app.Bookinfo(ns.Bookinfo))
		})
		ossm.BasicSetup(t)

		t.LogStep("Create cacerts secret for SMCP")
		oc.CreateGenericSecretFromFiles(t, meshNamespace, "cacerts",
			"ca-cert.pem="+cert.SampleCACert,
			"ca-key.pem="+cert.SampleCAKey,
			"root-cert.pem="+cert.SampleCARoot,
			"cert-chain.pem="+cert.SampleCAChain)

		smcp := ossm.DefaultClusterWideSMCP(t)
		smcp.Namespace = meshNamespace
		istio := ossm.DefaultIstio()
		istio.Template = istioCustomCATmpl
		istio.Namespace = meshNamespace

		t.LogStep("Deploy SMCP " + smcp.Version.String() + " with custom CA and SMMR")
		oc.ApplyTemplate(t, meshNamespace, serviceMeshCustomCATmpl, smcp)
		oc.WaitSMCPReady(t, meshNamespace, smcp.Name)

		t.LogStep("Verify that SMCP mutating webhook uses the custom CA")
		mutatingWebhookName := fmt.Sprintf("istiod-%s-%s", smcp.Name, meshNamespace)
		mutatingWebhookCABundle := oc.GetJson(t, "", "mutatingwebhookconfigurations", mutatingWebhookName, "{.webhooks[0].clientConfig.caBundle}")
		cacertsIntermediateCert := oc.GetJson(t, meshNamespace, "secrets", "cacerts", `{.data.ca-cert\.pem}`)
		if mutatingWebhookCABundle != cacertsIntermediateCert {
			t.Fatalf("Mutating Webhook '%s' caBundle does not match cacerts ca-cert.pem.\nwebhookBundle: %s\ncacertsCACert: %s\n", mutatingWebhookName, mutatingWebhookCABundle, cacertsIntermediateCert)
		}
		t.Log("SMCP mutating webhook caBundle matches cacerts ca-cert.pem")

		managedLabel := oc.GetJson(t, "", "mutatingwebhookconfigurations", mutatingWebhookName, `{.metadata.labels.maistra\.io/managed}`)
		if managedLabel != "true" {
			t.Fatalf("Mutating Webhook '%s' does not have maistra.io/managed=true label. Got: %s", mutatingWebhookName, managedLabel)
		}
		t.Log("SMCP mutating webhook has maistra.io/managed=true label")

		t.LogStep(`Add injection label to "bookinfo" namespace`)
		oc.Label(t, "", "Namespace", ns.Bookinfo, "istio-injection=enabled")

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

		// 3.0 uses the root cert for the validating webhook whereas 2.6 uses the intermediate cert.
		// When the 3.0 istiod begins to manage the webhook instead of the 2.6 operator,
		// the root cert should be used instead of the intermediate cert.
		t.LogStep("Ensure OSSM 3.0 validating webhook uses the custom CA root cert")
		validatingWebhookName := fmt.Sprintf("istio-validator-%s-%s", istio.Name, meshNamespace)
		cacertsRootCert := oc.GetJson(t, meshNamespace, "secrets", "cacerts", `{.data.root-cert\.pem}`)
		retry.UntilSuccess(t, func(t test.TestHelper) {
			validatingWebhookCABundle := oc.GetJson(t, "", "validatingwebhookconfigurations", validatingWebhookName, "{.webhooks[0].clientConfig.caBundle}")
			if validatingWebhookCABundle != cacertsRootCert {
				t.Errorf("Validating Webhook '%s' caBundle does not match cacerts root-cert.pem.\nwebhookBundle: %s\ncacertsRootCert: %s\n", validatingWebhookName, validatingWebhookCABundle, cacertsRootCert)
			}
		})
		t.Log("OSSM 3.0 validating webhook caBundle matches cacerts root-cert.pem")

		ensureResourceStable(t, validatingWebhookName, meshNamespace, "validatingwebhookconfigurations")

		t.LogStep("Migrate bookinfo to 3.0 controlplane")
		t.Log("Getting Istio active Rev name")
		ossm3RevName := oc.GetJson(t, "", "Istio", istio.Name, "{.status.activeRevisionName}")
		t.Log("Relabeling bookinfo namespace")
		oc.Label(t, "", "Namespace", ns.Bookinfo, maistraIgnoreLabel+" istio-injection- istio.io/rev="+ossm3RevName)
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

		curl.Request(t, bookinfoGatewayURL, nil, assert.RequestSucceeds("productpage request succeeded", "productpage request failed"))
	})
}
