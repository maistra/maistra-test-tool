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
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestMustGather(t *testing.T) {
	NewTest(t).Id("T30").Groups(Full, Disconnected).Run(func(t TestHelper) {
		t.Log("This test verifies must-gather log collection")

		ns := "bookinfo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		DeployControlPlane(t)

		t.LogStep("Deploy bookinfo in bookinfo ns")
		app.InstallAndWaitReady(t, app.Bookinfo(ns))

		image := env.GetMustGatherImage()
		dir := shell.CreateTempDir(t, "must-gather-")

		t.LogStepf("Capture must-gather using image %s without namespace flag", image)
		output := shell.Executef(t, `mkdir -p %s; oc adm must-gather --dest-dir=%s --image=%s`, dir, dir, "registry-proxy.engineering.redhat.com/rh-osbs/openshift-service-mesh-istio-must-gather-rhel8:2.4.0-11")
		if strings.Contains(output, "ERROR:") {
			t.Fatalf("Error was found during the execution of must-gather: %s\n", output)
		}

		t.NewSubTest("dump files and proxy stats files exist").Run(func(t test.TestHelper) {
			t.LogStep("Check files exist under the directory of mustgather: openshift-operators.servicemesh-resources.maistra.io.yaml, debug-syncz.json, config_dump_istiod.json, config_dump_proxy.json, proxy_stats")
			assertFilesExist(t, dir,
				"version",
				"openshift-operators.servicemesh-resources.maistra.io.yaml",
				"debug-syncz.json", 
				"config_dump_istiod.json",
				"config_dump_proxy.json",
				"proxy_stats")

			t.LogStep("verify content of proxy_stats")
			t.Log("verify that proxy stats file is not empty and contains parameters like: server.stats_recent_lookups, server.total_connections, server.uptime, server.version")
			proxyStatsFilePath := getPathToFile(t, dir, "proxy_stats")
			shell.Execute(t,
				fmt.Sprintf("cat %s", proxyStatsFilePath),
				assert.OutputContains("server.stats_recent_lookups",
					"server.stats_recent_lookups is on the proxy_stats file",
					"server.stats_recent_lookups is not on the proxy_stats file"),
				assert.OutputContains("server.total_connections",
					"server.total_connections is on the proxy_stats file",
					"server.total_connections is not on the proxy_stats file"),
				assert.OutputContains("server.uptime",
					"server.uptime is on the proxy_stats file",
					"server.uptime is not on the proxy_stats file"))
		})

		t.NewSubTest("version file exist").Run(func(t TestHelper) {
			t.LogStep("verify version exist")
			fileList := []string{"version"}
			verifyFilesExist(t, dir, fileList)

			t.LogStep("verify file version contains the version of the must-gather image")
			versionFilePath := getPathToFile(t, dir, "version")
			shell.Execute(t,
				fmt.Sprintf("cat %s", versionFilePath),
				assert.OutputContains(env.GetMustGatherTag(),
					"Expected must gather version was found",
					"Expected must gather version was not found"))
		})

		t.NewSubTest("resource cluster scoped exist").Run(func(t TestHelper) {
			t.LogStep("Verify resources for cluster scoped files are created")
			t.Log("verify that resources for cluster scoped are created: nodes, clusterrolebindings, clusterroles")
			clusterScopedResourcesPathList := []string{
				"cluster-scoped-resources/core/nodes/*.yaml",
				"cluster-scoped-resources/rbac.authorization.k8s.io/clusterrolebindings/*.yaml",
				"cluster-scoped-resources/rbac.authorization.k8s.io/clusterroles/*.yaml"}
			for _, path := range clusterScopedResourcesPathList {
				pathAndFilesExist(t, dir, path)
			}
		})

		t.NewSubTest("resource for namespaces exist").Run(func(t TestHelper) {
			t.LogStep("verify that resources for namespaces are created including bookinfo and istio-system folders")
			namespacedResourcesPathList := []string{
				"namespaces/istio-system/*.yaml",
				"namespaces/bookinfo/*.yaml",
				"namespaces/openshift-operators/openshift-operators.yaml",
				"namespaces/*/k8s.cni.cncf.io/network-attachment-definitions/*.yaml",
				"namespaces/*/rbac.authorization.k8s.io/rolebindings/*.yaml"}
			for _, path := range namespacedResourcesPathList {
				pathAndFilesExist(t, dir, path)
			}
		})

		t.NewSubTest("cluster service version files validation").Run(func(t TestHelper) {
			t.LogStep("Get service current service version from the cluster")
			csvList := shell.Execute(t, "oc get csv -n openshift-operators | awk 'NR>1 { print $1 }'")

			t.LogStep("verify if the csv files exist for the current service version")
			csvListSlice := strings.Split(csvList, "\n")
			for _, csv := range csvListSlice {
				if csv != "" {
					pathAndFilesExist(t,
						dir,
						fmt.Sprintf("namespaces/openshift-operators/operators.coreos.com/clusterserviceversions/%s.yaml", csv))
				}
			}
		})
	})
}

func verifyFilesExist(t TestHelper, dir string, fileList []string) {
	fileMatches := make(map[string]bool)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		for _, file := range fileList {
			match, _ := filepath.Match(file, info.Name())
			if match {
				fileMatches[file] = true
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Error: %s\n", err.Error())
	}

	for _, file := range fileList {
		if !fileMatches[file] {
			t.Fatalf("File not found: %s", file)
		}
	}
}

func pathAndFilesExist(t TestHelper, dir string, path string) {
	fullPath := filepath.Join(dir, path)
	pattern := filepath.Base(fullPath)
	pathExists := false
	filesFound := false

	err := filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		match, _ := filepath.Match(pattern, info.Name())
		if match {
			pathExists = true
			if !info.IsDir() {
				filesFound = true
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Error: %s\n", err.Error())
	}

	if !pathExists {
		t.Fatalf("Path not found: %s\n", path)
	}

	if !filesFound {
		t.Fatalf("No files found under path: %s\n", path)
	}
}

func getPathToFile(t TestHelper, dir string, fileName string) string {
	var pathToFile string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.Name() == fileName {
			pathToFile = path
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Error: %s\n", err.Error())
	}

	return pathToFile
}
