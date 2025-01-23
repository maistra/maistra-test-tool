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
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestMigrationSimpleClusterWide(t *testing.T) {
	test.NewTest(t).MinVersion(version.SMCP_2_6).Groups(test.Full, test.Migration).Run(func(t test.TestHelper) {
		
		// delete mesh namespace from previous tests
		t.LogStepf("Delete namespace %s", meshNamespace)
		oc.RecreateNamespace(t, meshNamespace)

		istio := ossm.DefaultIstio()
		t.Cleanup(func() {
			oc.DeleteResource(t, "", "Istio", istio.Name)
			oc.DeleteResource(t, "", "IstioCNI", "default")
			oc.DeleteNamespace(t, "istio-cni")
			// clean up bookinfo
			oc.DeleteFile(t, ns.Bookinfo, migrationGateway)
			app.Uninstall(t, app.Bookinfo(ns.Bookinfo))
			// need to delete SMCP since the next test can be in MultiTenant mode
			oc.RecreateNamespace(t, meshNamespace)
		})

		t.LogStep("Install SMCP 2.6 in clusterwide mode")
		smcp := ossm.DefaultClusterWideSMCP()
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

		t.LogStep("Install bookinfo and bookinfo gateway")
		app.InstallAndWaitReady(t, app.Bookinfo(ns.Bookinfo))

		// update default gateway
		oc.ApplyFile(t, ns.Bookinfo, migrationGateway)
		oc.DefaultOC.WaitDeploymentRolloutComplete(t, ns.Bookinfo, "bookinfo-gateway")
		oc.DefaultOC.WaitFor(t, ns.Bookinfo, "Route", "bookinfo-gateway", `jsonpath="{.status.ingress[].host}"`)
		hostname := oc.GetJson(t, ns.Bookinfo, "Routes", "bookinfo-gateway", "{.spec.host}")
		bookinfoGatewayURL := fmt.Sprintf("http://%s/productpage", hostname)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		continuallyRequest(ctx, t, bookinfoGatewayURL)

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

// Will continually request the URL until the context is cancelled and assert for success.
func continuallyRequest(ctx context.Context, t test.TestHelper, url string) {
	t.T().Helper()
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				t.Log("Ending continual requests. Context has been cancelled.")
				return
			case <-time.After(time.Second):
				curl.Request(t, url, curl.WithContext(ctx), assert.RequestSucceeds("productpage request succeeded", "productpage request failed"))
			default:
				if t.Failed() {
					t.Log("Ending continual requests. Test failed.")
					return
				}
			}
		}
	}(ctx)
}
