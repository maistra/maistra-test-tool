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
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

type ingressStatus struct {
	IP       string `json:"ip,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

func workloadNames(workloads []workload) []string {
	var names []string
	for _, wk := range workloads {
		names = append(names, wk.Name)
	}
	return names
}

type workload struct {
	Name   string
	Labels map[string]string
}

func toSelector(labels map[string]string) string {
	var parts []string
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}

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
		smcp := ossm.DefaultClusterWideSMCP()
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

		t.LogStep("Create 3.0 controlplane and IstioCNI")
		setupIstio(t, istio)

		t.LogStep("Install bookinfo and bookinfo gateway")
		app.InstallAndWaitReady(t, app.Bookinfo(ns.Bookinfo))

		// update default gateway
		oc.ApplyFile(t, ns.Bookinfo, migrationGateway)
		oc.DefaultOC.WaitDeploymentRolloutComplete(t, ns.Bookinfo, "bookinfo-gateway")
		oc.DefaultOC.WaitFor(t, ns.Bookinfo, "Route", "bookinfo-gateway", `jsonpath="{.status.ingress[].host}"`)
		hostname := oc.GetJson(t, ns.Bookinfo, "Routes", "bookinfo-gateway", "{.spec.host}")
		bookinfoGatewayURL := fmt.Sprintf("http://%s/productpage", hostname)
		continuallyRequest(t, bookinfoGatewayURL)

		t.LogStep("Migrate bookinfo to 3.0 controlplane")
		ossm3RevName := oc.GetJson(t, "", "Istio", istio.Name, "{.status.activeRevisionName}")
		oc.Label(t, "", "Namespace", ns.Bookinfo, "istio-injection- istio.io/rev="+ossm3RevName)
		// Wait for book info to be removed.
		retry.UntilSuccess(t, func(t test.TestHelper) {
			var members []string
			output := oc.GetJson(t, smcp.Namespace, "ServiceMeshMemberRoll", "default", "{.status.configuredMembers}")
			if err := json.Unmarshal([]byte(output), &members); err != nil {
				t.Error(err)
			}
			contains := false
			for _, member := range members {
				if member == "bookinfo" {
					contains = true
					break
				}
			}
			if contains {
				t.Error("bookinfo found in SMMR. Expected it to be removed.")
			}
		})
		oc.RestartAllPodsAndWaitReady(t, ns.Bookinfo)
		workloads := []string{
			"productpage-v1",
			"reviews-v1",
			"reviews-v2",
			"reviews-v3",
			"ratings-v1",
			"details-v1",
		}
		// Waiting for the rollouts to complete ensures that old pods have been deleted.
		// If there are old pods lying around then the assertion below to get the pod annotations
		// will fail.

		oc.WaitDeploymentRolloutComplete(t, ns.Bookinfo, workloads...)

		t.LogStep("Ensure all pods have migrated to 3.0 controlplane and curl requests succeed")
		for _, workload := range workloads {
			arr := strings.Split(workload, "-")
			app, version := arr[0], arr[1]
			annotations := oc.GetPodAnnotations(t, pod.MatchingSelector(fmt.Sprintf("app=%s,version=%s", app, version), ns.Bookinfo))
			if actual := annotations["istio.io/rev"]; actual != ossm3RevName {
				t.Fatalf("Expected %s. Got: %s", ossm3RevName, actual)
			}
		}

		// One last request to ensure bookinfo still works.
		curl.Request(t, bookinfoGatewayURL, nil, assert.RequestSucceeds("productpage request succeeded", "productpage request failed"))
	})
}

func TestMigrationSimpleClusterWideLoadBalancer(t *testing.T) {
	test.NewTest(t).MinVersion(version.SMCP_2_6).Groups(test.Migration).Run(func(t test.TestHelper) {
		if arch := env.GetArch(); arch == "p" || arch == "z" {
			t.Skipf("External LoadBalancer test not supported on arch: %s", arch)
		}
		if clusterSupportsIPv6(t) {
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
		smcp := ossm.DefaultClusterWideSMCP()
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

		t.LogStep("Create 3.0 controlplane and IstioCNI")
		setupIstio(t, istio)

		t.LogStep("Install bookinfo and bookinfo gateway")
		app.InstallAndWaitReady(t, app.Bookinfo(ns.Bookinfo))

		// update default gateway
		oc.ApplyFile(t, ns.Bookinfo, migrationGateway)
		oc.DefaultOC.WaitDeploymentRolloutComplete(t, ns.Bookinfo, "bookinfo-gateway")
		oc.DefaultOC.WaitFor(t, ns.Bookinfo, "Service", "bookinfo-gateway", `jsonpath='{.status.loadBalancer.ingress}'`)

		hostname := getLoadBalancerServiceHostname(t, "bookinfo-gateway", ns.Bookinfo)
		bookinfoGatewayURL := fmt.Sprintf("http://%s/productpage", hostname)

		continuallyRequest(t, bookinfoGatewayURL)

		t.LogStep("Migrate bookinfo to 3.0 controlplane")
		t.Log("Getting Istio active Rev name")
		ossm3RevName := oc.GetJson(t, "", "Istio", istio.Name, "{.status.activeRevisionName}")
		t.Log("Relabeling bookinfo namespace")
		oc.Label(t, "", "Namespace", ns.Bookinfo, "istio-injection- istio.io/rev="+ossm3RevName)
		// Wait for book info to be removed.
		retry.UntilSuccess(t, func(t test.TestHelper) {
			var members []string
			t.Log("Checking if \"bookinfo\" has been removed from default SMMR...")
			output := oc.GetJson(t, smcp.Namespace, "ServiceMeshMemberRoll", "default", "{.status.configuredMembers}")
			if err := json.Unmarshal([]byte(output), &members); err != nil {
				t.Error(err)
			}
			contains := false
			for _, member := range members {
				if member == "bookinfo" {
					contains = true
					break
				}
			}
			if contains {
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
			if output := oc.DefaultOC.Invokef(t, `kubectl get pods -n %s -o jsonpath='{.items[?(@.metadata.deletionTimestamp!="")].metadata.name}'`, ns.Bookinfo); output != "" {
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

// Will continually request the URL until the test has ended and assert for success.
// Once the test is over, this func will clean itself up and wait until in flight
// requests have finished.
func continuallyRequest(t test.TestHelper, url string) {
	t.T().Helper()
	t.Logf("Continually requesting URL: %s", url)

	ctx, cancel := context.WithCancel(context.Background())
	// 1. cancel
	// 2. wait for in flight requests to finish so that the curl assertion doesn't fail the test if underlying resources have been deleted.
	// 3. continue with other cleanups like deletion of resources.
	stopped := make(chan struct{})
	t.Cleanup(func() {
		cancel()
		t.Log("Waiting for continual requests to stop...")
		<-stopped
		t.Log("Continual requests stopped.")
	})
	go func(ctx context.Context) {
	ReqLoop:
		for {
			if t.Failed() {
				t.Log("Ending continual requests. Test failed.")
				break
			}

			select {
			case <-ctx.Done():
				t.Log("Ending continual requests. Context has been cancelled.")
				break ReqLoop
			case <-time.After(time.Second):
				curl.Request(t, url, curl.WithContext(ctx), assert.RequestSucceedsAndIgnoreContextCancelled("productpage request succeeded", "productpage request failed"))
				break ReqLoop
			}
		}
		stopped <- struct{}{}
	}(ctx)
}

// Detecting if the cluster supports ipv6 by looking for the kubernetes Service,
// which should always be there, and inspecting ipFamilies.
func clusterSupportsIPv6(t test.TestHelper) bool {
	t.T().Helper()
	var ipFamilies []string
	ipFamResp := oc.GetJson(t, ns.Default, "Service", "kubernetes", "{.spec.ipFamilies}")
	if err := json.Unmarshal([]byte(ipFamResp), &ipFamilies); err != nil {
		t.Fatalf("Unable to marshal ip family resp: %s", err)
	}

	for _, ipFamily := range ipFamilies {
		if ipFamily == "IPv6" {
			return true
		}
	}

	return false
}

func setupIstio(t test.TestHelper, istio ossm.Istio) {
	t.T().Helper()
	t.Cleanup(func() {
		oc.DeleteResource(t, "", "Istio", istio.Name)
		oc.DeleteResource(t, "", "IstioCNI", "default")
	})
	istioTempl := `apiVersion: sailoperator.io/v1alpha1
kind: Istio
metadata:
  name: {{ .Name }}
spec:
  namespace: {{ .Namespace }}
  version: {{ .Version }}`
	oc.ApplyTemplate(t, "", istioTempl, istio)
	oc.DefaultOC.WaitFor(t, "", "Istio", istio.Name, "condition=Ready")
	oc.CreateNamespace(t, "istio-cni")
	istioCNI := `apiVersion: sailoperator.io/v1alpha1
kind: IstioCNI
metadata:
  name: default
spec:
  namespace: istio-cni
  version: {{ .Version }}`
	oc.ApplyTemplate(t, "", istioCNI, istio)
	oc.DefaultOC.WaitFor(t, "", "IstioCNI", "default", "condition=Ready")
}

// Returns either the ip address or the hostname of the LoadBalancer from the Service status.
// Fails if neither exist.
func getLoadBalancerServiceHostname(t test.TestHelper, name string, namespace string) string {
	t.T().Helper()
	resp := oc.GetJson(t, ns.Bookinfo, "Service", "bookinfo-gateway", "{.status.loadBalancer.ingress}")
	var v []ingressStatus
	if err := json.Unmarshal([]byte(resp), &v); err != nil {
		t.Fatalf("Unable to unmarshal ingress status from Service response: %s", err)
	}
	if got := len(v); got != 1 {
		t.Fatalf("Expected there to be a 1 ingress but there are: %d", got)
	}
	status := v[0]

	var hostname string
	if status.IP != "" {
		hostname = status.IP
	} else if status.Hostname != "" {
		hostname = status.Hostname
	} else {
		t.Fatalf("Service: %s/%s has neither an ip or hostname", name, namespace)
	}

	return hostname
}
