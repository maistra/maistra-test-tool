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
	"encoding/json"
	"fmt"
	"strings"
	"testing"

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
	test.NewTest(t).MinVersion(version.SMCP_2_6).Groups(test.Full).Run(func(t test.TestHelper) {
		t.Cleanup(func() {
			oc.DeleteResource(t, "", "Istio", "example")
			oc.DeleteResource(t, "", "IstioCNI", "default")
		})
		defaults := ossm.DefaultClusterWideSMCP()
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
		oc.ApplyTemplate(t, defaults.Namespace, templ, defaults)

		oc.WaitSMCPReady(t, defaults.Namespace, defaults.Name)
		oc.WaitSMMRReady(t, defaults.Namespace)
		oc.Label(t, "", "Namespace", ns.Bookinfo, "istio-injection=enabled")
		t.LogStep("Update SMMR to include bookinfo namespace")
		// Wait for SMMR to include bookinfo
		oc.DefaultOC.WaitFor(t, defaults.Namespace, "ServiceMeshMemberRoll", "default", `jsonpath='{.status.configuredMembers[?(@=="bookinfo")]}'`)

		t.LogStep("Create 3.0 controlplane and IstioCNI")
		istio := `apiVersion: sailoperator.io/v1alpha1
kind: Istio
metadata:
  name: example
spec:
  namespace: istio-system
  version: v1.24.1`
		oc.ApplyString(t, "", istio)
		oc.DefaultOC.WaitFor(t, "", "Istio", "example", "condition=Ready")
		oc.CreateNamespace(t, "istio-cni")
		istioCNI := `apiVersion: sailoperator.io/v1alpha1
kind: IstioCNI
metadata:
  name: default
spec:
  namespace: istio-cni
  version: v1.24.1`
		oc.ApplyString(t, "", istioCNI)

		t.LogStep("Install bookinfo and bookinfo gateway")
		app.InstallAndWaitReady(t, app.Bookinfo(ns.Bookinfo))

		oc.ApplyFile(t, ns.Bookinfo, "bookinfo-gateway.yaml")
		oc.DefaultOC.WaitDeploymentRolloutComplete(t, "bookinfo", "bookinfo-gateway")
		ip := oc.DefaultOC.Invokef(t, "kubectl get service -n %s %s -o jsonpath='{.status.loadBalancer.ingress[0].ip}'", ns.Bookinfo, "bookinfo-gateway")
		bookinfoGatewayURL := fmt.Sprintf("http://%s/productpage", ip)
		curl.Request(t, bookinfoGatewayURL, nil, assert.RequestSucceeds("productpage request succeeded", "productpage request failed"))

		t.LogStep("Migrate bookinfo to 3.0 controlplane")
		oc.Label(t, "", "Namespace", ns.Bookinfo, "istio-injection- istio.io/rev=example")
		// Wait for book info to be removed.
		retry.UntilSuccess(t, func(t test.TestHelper) {
			var members []string
			output := oc.GetJson(t, defaults.Namespace, "ServiceMeshMemberRoll", "default", "{.status.configuredMembers}")
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
			if actual := annotations["istio.io/rev"]; actual != "example" {
				t.Fatalf("Expected example. Got: %s", actual)
			}
		}

		curl.Request(t, bookinfoGatewayURL, nil, assert.RequestSucceeds("productpage request succeeded", "productpage request failed"))
	})
}
