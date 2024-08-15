// Copyright 2021 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ossm

import (
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/istioctl"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/template"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestDiscoverySelectors(t *testing.T) {
	NewTest(t).Groups(Full, ARM).Run(func(t TestHelper) {
		t.Log("This test checks if discoverySelectors are being honored")
		t.Log("See https://issues.redhat.com/browse/OSSM-3802")
		t.Log("Test case is based on https://istio.io/latest/blog/2021/discovery-selectors/")
		if env.GetSMCPVersion().LessThan(version.SMCP_2_4) {
			t.Skip("Skipped because spec.meshConfig.discoverySelectors is only available in v2.4+")
		}

		t.LogStep("Apply cluster-wide SMCP and standard SMMR")
		oc.RecreateNamespace(t, meshNamespace)
		oc.ApplyString(t, meshNamespace, smmr)
		oc.ApplyString(t, meshNamespace, template.Run(t, clusterWideSMCP, DefaultSMCP()))
		oc.WaitSMCPReady(t, meshNamespace, DefaultSMCP().Name)
		oc.WaitSMMRReady(t, meshNamespace)

		t.LogStep("Install httpbin and sleep pod")
		app.InstallAndWaitReady(t, app.Sleep(ns.Foo), app.Httpbin(ns.MeshExternal))
		t.Cleanup(func() {
			app.Uninstall(t, app.Sleep(ns.Foo), app.Httpbin(ns.MeshExternal))
		})

		t.LogStep("Confirm that the httpbin and sleep services have been discovered")
		istioctl.CheckClusters(t,
			pod.MatchingSelector("app=sleep", ns.Foo),
			assert.OutputContains("sleep",
				"Sleep was discovered",
				"Expected sleep to be discovered, but it was not found"),
			assert.OutputContains("httpbin",
				"Httpbin was discovered",
				"Expected Httpbin to be discovered, but it was not found."))

		t.LogStep("Configure discoverySelectors so that only namespace foo is discovered")
		oc.Label(t, "", "namespace", ns.Foo, "istio-discovery=enabled")
		oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
spec:
  meshConfig:
    discoverySelectors:
    - matchLabels:
        istio-discovery: enabled`)

		t.Cleanup(func() {
			oc.Patch(t, meshNamespace,
				"smcp", smcpName,
				"json",
				`[{"op": "remove", "path": "/spec/meshConfig"}]`)
		})

		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Verify that sleep service has been discovered, whereas httpbin hasn't")
		istioctl.CheckClusters(t,
			pod.MatchingSelector("app=sleep", ns.Foo),
			assert.OutputContains("sleep",
				"Sleep was discovered",
				"Expected sleep to be discovered, but it was not found"),
			assert.OutputDoesNotContain("httpbin",
				"Httpbin was not discovered",
				"Expected Httpbin to not be discovered, but it was."))
	})
}

const (
	clusterWideSMCP = `
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
spec:
  version: {{ .Version }}
  mode: ClusterWide
  tracing:
    type: None
  addons:
    grafana:
      enabled: false
    kiali:
      enabled: false
    prometheus:
      enabled: false
  {{ if .Rosa }} 
  security:
    identity:
      type: ThirdParty
  {{ end }}`
)
