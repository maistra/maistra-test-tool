// Copyright 2024 Red Hat, Inc.
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
	"net"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/check/require"
	"github.com/maistra/maistra-test-tool/pkg/util/cluster"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestMigrationSimpleClusterWide(t *testing.T) {
	test.NewTest(t).MinVersion(version.SMCP_2_6).Groups(test.Migration).Run(func(t test.TestHelper) {
		// delete mesh namespace from previous tests
		t.LogStep("Cleanup any lingering namespaces from previous runs")
		oc.DeleteTestBoundNamespaces(t)
		oc.CreateNamespace(t, meshNamespace)

		t.Cleanup(func() {
			oc.DeleteTestBoundNamespaces(t)
			// clean up bookinfo
			oc.DeleteFile(t, ns.Bookinfo, migrationGateway)
			app.Uninstall(t, app.Bookinfo(ns.Bookinfo))
		})

		t.LogStep("Install SMCP 2.6 in clusterwide mode")
		smcp := ossm.DefaultClusterWideSMCP(t)
		istio := ossm.DefaultIstio()
		// These are defaulted to the same but better to be explicit.
		istio.Namespace = smcp.Namespace
		ossm.BasicSetup(t)
		templ := `apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
spec:
  version: {{ .Version }}
  tracing:
    type: None
  security:
    manageNetworkPolicy: false
  gateways:
    enabled: false
  policy:
    type: Istiod
  addons:
    grafana:
      enabled: false
    kiali:
      enabled: false
    prometheus:
      enabled: false
  mode: ClusterWide`
		oc.ApplyTemplate(t, smcp.Namespace, templ, smcp)

		oc.WaitSMCPReady(t, smcp.Namespace, smcp.Name)
		// Need to add the injection label first so that the SMMR gets created.
		// SMMR will only get created if a namespace has an injection label.
		oc.Label(t, "", "Namespace", ns.Bookinfo, "istio-injection=enabled")
		oc.WaitSMMRReady(t, smcp.Namespace)
		// Wait for SMMR to include bookinfo
		oc.DefaultOC.WaitFor(t, smcp.Namespace, "ServiceMeshMemberRoll", "default", `jsonpath='{.status.configuredMembers[?(@=="bookinfo")]}'`)

		t.Log("Enable strict mTLS for the whole mesh")
		oc.ApplyString(t, smcp.Namespace, enableMTLSPeerAuth)

		t.LogStep("Install bookinfo and bookinfo gateway")
		app.InstallAndWaitReady(t, app.Bookinfo(ns.Bookinfo))

		// update default gateway
		oc.ApplyFile(t, ns.Bookinfo, migrationGateway)
		oc.DefaultOC.WaitDeploymentRolloutComplete(t, ns.Bookinfo, "bookinfo-gateway")
		oc.DefaultOC.WaitFor(t, ns.Bookinfo, "Route", "bookinfo-gateway", `jsonpath="{.status.ingress[].host}"`)
		hostname := oc.GetJson(t, ns.Bookinfo, "Routes", "bookinfo-gateway", "{.spec.host}")
		bookinfoGatewayURL := fmt.Sprintf("http://%s/productpage", hostname)
		continuallyRequest(t, bookinfoGatewayURL)

		t.LogStep("Create 3.0 controlplane and IstioCNI")
		setupIstio(t, istio)

		t.LogStep("Migrate bookinfo to 3.0 controlplane")
		ossm3RevName := oc.GetJson(t, "", "Istio", istio.Name, "{.status.activeRevisionName}")
		oc.Label(t, "", "Namespace", ns.Bookinfo, maistraIgnoreLabel+" istio-injection- istio.io/rev="+ossm3RevName)
		// Wait for book info to be removed.
		retry.UntilSuccess(t, func(t test.TestHelper) {
			t.Log("Checking if \"bookinfo\" has been removed from default SMMR...")
			if namespaceInSMMR(t, ns.Bookinfo, "default", smcp.Namespace) {
				t.Error("bookinfo found in SMMR. Expected it to be removed.")
			}
		})

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

func TestMigrationSimpleMultiTenant(t *testing.T) {
	test.NewTest(t).MinVersion(version.SMCP_2_6).Groups(test.Migration).Run(func(t test.TestHelper) {
		// delete mesh namespace from previous tests
		t.LogStep("Cleanup resources from previous runs")
		oc.DeleteTestBoundNamespaces(t)
		oc.DeleteResourcesByLabel(t, "", "Istio", oc.MaistraTestLabel)

		t.Cleanup(func() {
			oc.DeleteTestBoundNamespaces(t)
		})

		t.LogStep("Install SMCPs for tenant-a and tenant-b in multitenant mode")

		smcpA := ossm.DefaultSMCP(t)
		smcpA.Namespace = "tenant-a"
		oc.CreateNamespace(t, smcpA.Namespace)

		smcpB := ossm.DefaultSMCP(t)
		smcpB.Namespace = "tenant-b"
		oc.CreateNamespace(t, smcpB.Namespace)

		istioA := ossm.DefaultIstio()
		istioA.Namespace = smcpA.Namespace
		istioA.Name = smcpA.Namespace

		istioB := ossm.DefaultIstio()
		istioB.Namespace = smcpB.Namespace
		istioB.Name = smcpB.Namespace

		const (
			bookinfoA = "bookinfo-a"
			bookinfoB = "bookinfo-b"
		)
		installSMCPWithBookinfo(t, smcpA, bookinfoA)
		installSMCPWithBookinfo(t, smcpB, bookinfoB)

		t.LogStep("Create 3.0 controlplane and IstioCNI")
		istioA.Template = `apiVersion: sailoperator.io/v1
kind: Istio
metadata:
  name: {{ .Name }}
spec:
  namespace: {{ .Namespace }}
  values:
    meshConfig:
      outboundTrafficPolicy:
        mode: REGISTRY_ONLY
      discoverySelectors:
        - matchLabels:
            tenant: tenant-a`
		istioB.Template = `apiVersion: sailoperator.io/v1
kind: Istio
metadata:
  name: {{ .Name }}
spec:
  namespace: {{ .Namespace }}
  values:
    meshConfig:
      outboundTrafficPolicy:
        mode: REGISTRY_ONLY
      discoverySelectors:
        - matchLabels:
            tenant: tenant-b`
		setupIstio(t, istioA, istioB)

		t.LogStep("Migrate bookinfo A to 3.0 controlplane")
		migrateBookinfo(t, istioA, bookinfoA)

		t.LogStep("Migrate bookinfo B to 3.0 controlplane")
		migrateBookinfo(t, istioB, bookinfoB)
	})
}

func migrateBookinfo(t test.TestHelper, istio ossm.Istio, bookinfoNamespace string) {
	ossm3RevName := oc.GetJson(t, "", "Istio", istio.Name, "{.status.activeRevisionName}")
	oc.Label(t, "", "Namespace", bookinfoNamespace, fmt.Sprintf("%s istio.io/rev=%s tenant=%s", maistraIgnoreLabel, ossm3RevName, ossm3RevName))

	workloads := []workload{
		{Name: "productpage-v1", Labels: map[string]string{"app": "productpage", "version": "v1"}},
		{Name: "reviews-v1", Labels: map[string]string{"app": "reviews", "version": "v1"}},
		{Name: "reviews-v2", Labels: map[string]string{"app": "reviews", "version": "v2"}},
		{Name: "reviews-v3", Labels: map[string]string{"app": "reviews", "version": "v3"}},
		{Name: "ratings-v1", Labels: map[string]string{"app": "ratings", "version": "v1"}},
		{Name: "details-v1", Labels: map[string]string{"app": "details", "version": "v1"}},
		{Name: "bookinfo-gateway", Labels: map[string]string{"istio": "bookinfo-gateway"}},
	}
	oc.DefaultOC.RestartDeployments(t, bookinfoNamespace, workloadNames(workloads)...)
	// Waiting for the rollouts to complete ensures that old pods have been deleted.
	// If there are old pods lying around then the assertion below to get the pod annotations
	// will fail.
	oc.WaitDeploymentRolloutComplete(t, bookinfoNamespace, workloadNames(workloads)...)
	retry.UntilSuccess(t, func(t test.TestHelper) {
		if output := oc.DefaultOC.Invokef(t, `oc get pods -n %s -o jsonpath='{.items[?(@.metadata.deletionTimestamp!="")].metadata.name}'`, bookinfoNamespace); output != "" {
			t.Errorf("Pods still being deleted: %s", output)
		}
	})

	t.LogStep("Ensure all pods have migrated to 3.0 controlplane and curl requests succeed")
	for _, workload := range workloads {
		annotations := oc.GetPodAnnotations(t, pod.MatchingSelector(toSelector(workload.Labels), bookinfoNamespace))
		if actual := annotations["istio.io/rev"]; actual != ossm3RevName {
			t.Fatalf("Expected %s. Got: %s", ossm3RevName, actual)
		}
	}

	// The curl request can return 200 to productpage but the downstream apps like reviews may be returning errors.
	// To ensure that all the apps are actually in the istio registry, we're going to check the debug endpoint
	// for the controlplane.
	checks := []common.CheckFunc{
		require.OutputContains(fmt.Sprintf(`"reviews.%s.svc.cluster.local"`, bookinfoNamespace), "Registry contains reviews app", "Registry missing reviews app"),
		require.OutputContains(fmt.Sprintf(`"productpage.%s.svc.cluster.local"`, bookinfoNamespace), "Registry contains productpage app", "Registry missing productpage app"),
		require.OutputContains(fmt.Sprintf(`"details.%s.svc.cluster.local"`, bookinfoNamespace), "Registry contains details app", "Registry missing details app"),
		require.OutputContains(fmt.Sprintf(`"ratings.%s.svc.cluster.local"`, bookinfoNamespace), "Registry contains ratings app", "Registry missing ratings app"),
	}
	oc.Exec(t, pod.MatchingSelectorFirst("istio.io/rev="+istio.Name, istio.Namespace), "discovery", "curl localhost:15014/debug/registryz", checks...)

	hostname := oc.GetJson(t, bookinfoNamespace, "Routes", "bookinfo-gateway", "{.spec.host}")
	bookinfoGatewayURL := fmt.Sprintf("http://%s/productpage", hostname)

	// One last request to ensure bookinfo still works.
	curl.Request(t, bookinfoGatewayURL, nil, assert.RequestSucceeds("productpage request succeeded", "productpage request failed"))
}

func installSMCPWithBookinfo(t test.TestHelper, smcp ossm.SMCP, bookinfoNamespace string) {
	t.T().Helper()
	oc.CreateNamespace(t, bookinfoNamespace)

	templ := `apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
spec:
  version: {{ .Version }}
  tracing:
    type: None
  security:
    manageNetworkPolicy: false
  gateways:
    enabled: false
  policy:
    type: Istiod
  addons:
    grafana:
      enabled: false
    kiali:
      enabled: false
    prometheus:
      enabled: false
  mode: MultiTenant`
	oc.ApplyTemplate(t, smcp.Namespace, templ, smcp)
	oc.WaitSMCPReady(t, smcp.Namespace, smcp.Name)

	bookinfoATmpl := fmt.Sprintf(`apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:
    - %s
`, bookinfoNamespace)
	oc.ApplyString(t, smcp.Namespace, bookinfoATmpl)
	oc.WaitSMMRReady(t, smcp.Namespace)
	// Wait for SMMR to include bookinfo
	oc.DefaultOC.WaitFor(t, smcp.Namespace, "ServiceMeshMemberRoll", "default", fmt.Sprintf(`jsonpath='{.status.configuredMembers[?(@=="%s")]}'`, bookinfoNamespace))

	t.Logf("Enable strict mTLS for mesh %s", smcp.Name)
	oc.ApplyString(t, smcp.Namespace, enableMTLSPeerAuth)

	t.Logf("Install bookinfo app and bookinfo gateway in namespace %s", bookinfoNamespace)
	app.InstallAndWaitReady(t, app.Bookinfo(bookinfoNamespace))
	app.InstallAndWaitReady(t, app.Bookinfo(bookinfoNamespace))

	// update default gateway
	oc.ApplyFile(t, bookinfoNamespace, migrationGateway)
	oc.DefaultOC.WaitDeploymentRolloutComplete(t, bookinfoNamespace, "bookinfo-gateway")
	oc.DefaultOC.WaitFor(t, bookinfoNamespace, "Route", "bookinfo-gateway", `jsonpath="{.status.ingress[].host}"`)
	hostname := oc.GetJson(t, bookinfoNamespace, "Routes", "bookinfo-gateway", "{.spec.host}")
	bookinfoGatewayURL := fmt.Sprintf("http://%s/productpage", hostname)
	continuallyRequest(t, bookinfoGatewayURL)
}

func TestMigrationSimpleClusterWideLoadBalancer(t *testing.T) {
	test.NewTest(t).MinVersion(version.SMCP_2_6).Groups(test.Migration).Run(func(t test.TestHelper) {
		if arch := env.GetArch(); arch == "p" || arch == "z" {
			t.Skipf("External LoadBalancer test not supported on arch: %s", arch)
		}
		if cluster.SupportsIPv6(t) {
			t.Skip("External LoadBalancer test not supported on IPv6 cluster")
		}

		// delete mesh namespace from previous tests
		t.LogStep("Cleanup any lingering namespaces from previous runs")
		oc.DeleteTestBoundNamespaces(t)
		oc.CreateNamespace(t, meshNamespace)

		t.Cleanup(func() {
			oc.DeleteTestBoundNamespaces(t)
			// clean up bookinfo
			oc.DeleteFile(t, ns.Bookinfo, migrationGateway)
			app.Uninstall(t, app.Bookinfo(ns.Bookinfo))
		})

		t.LogStep("Install SMCP 2.6 in clusterwide mode")
		smcp := ossm.DefaultClusterWideSMCP(t)
		istio := ossm.DefaultIstio()
		// These are defaulted to the same but better to be explicit.
		istio.Namespace = smcp.Namespace
		ossm.BasicSetup(t)
		templ := `apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
spec:
  version: {{ .Version }}
  tracing:
    type: None
  security:
    manageNetworkPolicy: false
  gateways:
    enabled: false
  policy:
    type: Istiod
  addons:
    grafana:
      enabled: false
    kiali:
      enabled: false
    prometheus:
      enabled: false
  mode: ClusterWide`
		oc.ApplyTemplate(t, smcp.Namespace, templ, smcp)

		oc.WaitSMCPReady(t, smcp.Namespace, smcp.Name)
		// Need to add the injection label first so that the SMMR gets created.
		// SMMR will only get created if a namespace has an injection label.
		oc.Label(t, "", "Namespace", ns.Bookinfo, "istio-injection=enabled")
		oc.WaitSMMRReady(t, smcp.Namespace)
		// Wait for SMMR to include bookinfo
		oc.DefaultOC.WaitFor(t, smcp.Namespace, "ServiceMeshMemberRoll", "default", `jsonpath='{.status.configuredMembers[?(@=="bookinfo")]}'`)

		t.Log("Enable strict mTLS for the whole mesh")
		oc.ApplyString(t, smcp.Namespace, enableMTLSPeerAuth)

		t.LogStep("Install bookinfo and bookinfo gateway")
		app.InstallAndWaitReady(t, app.Bookinfo(ns.Bookinfo))

		// update default gateway
		oc.ApplyFile(t, ns.Bookinfo, migrationGateway)
		oc.DefaultOC.WaitDeploymentRolloutComplete(t, ns.Bookinfo, "bookinfo-gateway")
		oc.DefaultOC.WaitFor(t, ns.Bookinfo, "Service", "bookinfo-gateway", `jsonpath='{.status.loadBalancer.ingress}'`)

		ingress := getLoadBalancerServiceHostname(t, "bookinfo-gateway", ns.Bookinfo)
		// In some clouds, namely AWS, it can take a minute for the DNS name to propagate after it's assigned to the LB.
		if hostname := ingress.Hostname; hostname != "" {
			// Wait 8 minutes altogether... it can take awhile.
			retry.UntilSuccessWithOptions(t, retry.Options().DelayBetweenAttempts(time.Second*4).MaxAttempts(120), func(t test.TestHelper) {
				addrs, err := net.LookupHost(hostname)
				if err != nil {
					t.Error(err)
					return
				}
				if len(addrs) == 0 {
					t.Errorf("No addresses found for host: %s", hostname)
					return
				}
			})
		}
		bookinfoGatewayURL := fmt.Sprintf("http://%s/productpage", ingress.GetHostname())

		continuallyRequest(t, bookinfoGatewayURL)

		t.LogStep("Create 3.0 controlplane and IstioCNI")
		setupIstio(t, istio)

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
