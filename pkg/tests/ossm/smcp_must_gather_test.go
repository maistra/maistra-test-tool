// Copyright Red Hat, Inc.
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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/version"

	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestMustGather(t *testing.T) {
	NewTest(t).Id("T30").Groups(Full, ARM).Run(func(t TestHelper) {
		t.Log("This test verifies must-gather log collection")

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns.Bookinfo)
		})

		DeployControlPlane(t)

		t.LogStep("Deploy bookinfo in bookinfo ns")
		app.InstallAndWaitReady(t, app.Bookinfo(ns.Bookinfo))

		image := env.GetMustGatherImage()
		dir := shell.CreateTempDir(t, "must-gather-")

		t.LogStepf("Capture must-gather using image %s without namespace flag", image)
		shell.Execute(t,
			fmt.Sprintf(`oc adm must-gather --dest-dir=%s --image=%s`, dir, image),
			assert.OutputDoesNotContain(
				"ERROR:",
				"Must gather completed successfully",
				"Must gather failed"))

		t.NewSubTest("dump files and proxy stats files exist for pods").Run(func(t TestHelper) {
			t.LogStep("Check dump files exist under the directory of namespace directory")
			t.Log("Verify these files:")
			t.Log("config_dump_istiod.json, config_dump_proxy.json, proxy_stats")
			assertFilesExist(t, dir,
				"**/namespaces/bookinfo/pods/details*/config_dump_istiod.json",
				"**/namespaces/bookinfo/pods/details*/config_dump_proxy.json",
				"**/namespaces/bookinfo/pods/details*/proxy_stats")

			t.LogStep("verify content of proxy_stats")
			t.Log("verify that proxy stats file is not empty and contains parameters like: server.stats_recent_lookups, server.total_connections, server.uptime, server.version")
			proxyStatsContent := []string{"server.stats_recent_lookups", "server.total_connections", "server.uptime", "server.version"}
			for _, content := range proxyStatsContent {
				checkFileContents(t,
					dir,
					"**/namespaces/bookinfo/pods/details*/proxy_stats",
					assert.OutputContains(
						content,
						fmt.Sprintf("%s is on the proxy_stats file", content),
						fmt.Sprintf("%s is not on the proxy_stats file", content)))
			}
		})

		t.NewSubTest("version file").Run(func(t TestHelper) {
			t.LogStep("verify file version exists")
			assertFilesExist(t, dir,
				"**/version")

			t.LogStep("verify file version contains the version of the must-gather image")
			checkFileContents(t,
				dir,
				"**/version",
				assert.OutputContains(
					env.GetMustGatherTag(),
					"Expected must gather version was found",
					"Expected must gather version was not found"))
		})

		t.NewSubTest("resources cluster scoped").Run(func(t TestHelper) {
			t.LogStep("Get nodes of the cluster")
			nodeOutput := shell.Execute(t, "oc get nodes | awk 'NR>1 { print $1 }'")
			nodeSlice := strings.Split(nodeOutput, "\n")

			t.LogStep("verify nodes files exist in cluster-scoped-resources")
			for _, node := range nodeSlice {
				if node != "" {
					assertFilesExist(t,
						dir,
						fmt.Sprintf("**/cluster-scoped-resources/core/nodes/%s.yaml", node))
				}
			}

			t.LogStep("Verify cluster-scoped-resources files exist in cluster-scoped-resources folder")
			if env.GetOperatorVersion().LessThan(version.OPERATOR_2_6_0) {
				assertFilesExist(t,
					dir,
					"**/cluster-scoped-resources/rbac.authorization.k8s.io/clusterrolebindings/istiod-internal-basic-istio-system.yaml",
					"**/cluster-scoped-resources/admissionregistration.k8s.io/mutatingwebhookconfigurations/openshift-operators.servicemesh-resources.maistra.io.yaml",
					"**/cluster-scoped-resources/admissionregistration.k8s.io/mutatingwebhookconfigurations/istiod-basic-istio-system.yaml",
					"**/cluster-scoped-resources/admissionregistration.k8s.io/validatingwebhookconfigurations/openshift-operators.servicemesh-resources.maistra.io.yaml",
					"**/cluster-scoped-resources/admissionregistration.k8s.io/validatingwebhookconfigurations/istio-validator-basic-istio-system.yaml",
					"**/cluster-scoped-resources/rbac.authorization.k8s.io/clusterroles/istiod-clusterrole-basic-istio-system.yaml")
			} else {
				assertFilesExist(t,
					dir,
					"**/cluster-scoped-resources/rbac.authorization.k8s.io/clusterrolebindings/istiod-internal-basic-istio-system.yaml",
					"**/cluster-scoped-resources/admissionregistration.k8s.io/mutatingwebhookconfigurations/istiod-basic-istio-system.yaml",
					"**/cluster-scoped-resources/admissionregistration.k8s.io/validatingwebhookconfigurations/istio-validator-basic-istio-system.yaml",
					"**/cluster-scoped-resources/rbac.authorization.k8s.io/clusterroles/istiod-clusterrole-basic-istio-system.yaml")

				webhookMap := map[string]string{
					"smcp.mutation.maistra.io":   "mutatingwebhookconfigurations",
					"smmr.mutation.maistra.io":   "mutatingwebhookconfigurations",
					"smcp.validation.maistra.io": "validatingwebhookconfigurations",
					"smmr.validation.maistra.io": "validatingwebhookconfigurations",
					"smm.validation.maistra.io":  "validatingwebhookconfigurations",
				}

				for webhook, kind := range webhookMap {
					name := oc.GetResouceNameByLabel(t, "", kind, fmt.Sprintf("olm.webhook-description-generate-name=%s", webhook))
					filename := fmt.Sprintf("%s.yaml", name)
					assertFilesExist(t,
						dir,
						fmt.Sprintf("**/cluster-scoped-resources/admissionregistration.k8s.io/%s/%s", kind, filename))
				}
			}
		})

		t.NewSubTest("resource for namespaces exist").Run(func(t TestHelper) {
			t.LogStep("verify that resources for namespaces are created including bookinfo and istio-system folders")
			assertFilesExist(t,
				dir,
				"**/namespaces/istio-system/debug-syncz.json",
				"**/namespaces/istio-system/istio-system.yaml",
				"**/namespaces/bookinfo/bookinfo.yaml",
				"**/namespaces/openshift-operators/openshift-operators.yaml",
				"**/namespaces/*/rbac.authorization.k8s.io/rolebindings/mesh-users.yaml")
		})

		t.NewSubTest("cluster service version files validation").Run(func(t TestHelper) {
			t.LogStep("Get service current service version from the cluster")
			csvList := shell.Execute(t, "oc get csv -n openshift-operators | awk 'NR>1 { print $1 }'")

			t.LogStep("verify if the csv files exist for the current service version")
			csvListSlice := strings.Split(csvList, "\n")
			for _, csv := range csvListSlice {
				if csv != "" {
					assertFilesExist(t,
						dir,
						fmt.Sprintf("**/namespaces/openshift-operators/operators.coreos.com/clusterserviceversions/%s.yaml", csv))
				}
			}
		})
	})
}

func assertFilesExist(t TestHelper, dir string, files ...string) {
	for _, file := range files {
		filePath := filepath.Join(dir, file)
		pathSplit := strings.Split(filePath, "/")
		fileName := pathSplit[len(pathSplit)-1]

		shell.Execute(t,
			fmt.Sprintf("find %s", filePath),
			assert.OutputContains(
				fileName,
				fmt.Sprintf("%s exists", filePath),
				fmt.Sprintf("%s does not exist", filePath)))
	}
}

func checkFileContents(t TestHelper, dir string, file string, checks ...common.CheckFunc) {
	path := filepath.Join(dir, file)
	filePath := shell.Execute(t, fmt.Sprintf("find %s", path))
	data, err := os.ReadFile(filePath[:len(filePath)-1])
	if err != nil {
		t.Fatalf("failed to read file: %s", err)
	}

	proxyStatsFileContent := string(data)
	if proxyStatsFileContent == "" {
		t.Fatalf("proxy_stats file is empty")
	}

	for _, check := range checks {
		check(t, proxyStatsFileContent)
	}
}
