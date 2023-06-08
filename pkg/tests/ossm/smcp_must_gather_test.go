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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
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

		t.NewSubTest("run must gather and verify files exists").Run(func(t TestHelper) {
			t.LogStepf("Capture must-gather using image %s", image)
			output := shell.Executef(t, `mkdir -p %s; oc adm must-gather --dest-dir=%s --image=%s`, dir, dir, "registry-proxy.engineering.redhat.com/rh-osbs/openshift-service-mesh-istio-must-gather-rhel8:2.4.0-11")
			if strings.Contains(output, "ERROR:") {
				t.Fatalf("Error was found during the execution of must-gather: %s\n", output)
			}

			t.LogStep("Check files exist under the directory of mustgather: openshift-operators.servicemesh-resources.maistra.io.yaml, debug-syncz.json, config_dump_istiod.json, config_dump_proxy.json, proxy_stats")
			fileList := []string{"openshift-operators.servicemesh-resources.maistra.io.yaml", "debug-syncz.json", "config_dump_istiod.json", "config_dump_proxy.json", "proxy_stats"}
			verifyFilesExist(t, dir, fileList)
		})
	})
}

func verifyFilesExist(t TestHelper, dir string, fileList []string) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		for _, file := range fileList {
			match, _ := filepath.Match(file, info.Name())
			if match {
				t.Logf("File found: %s\n", path)
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Error: %s\n", err.Error())
	}
}
