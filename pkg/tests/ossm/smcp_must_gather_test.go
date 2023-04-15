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
	"path/filepath"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestMustGather(t *testing.T) {
	NewTest(t).Id("T30").Groups(Full).Run(func(t TestHelper) {
		t.Log("This test verifies must-gather log collection")

		ns := "bookinfo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		t.LogStep("Deploy bookinfo in bookinfo ns")
		app.InstallAndWaitReady(t, app.Bookinfo(ns))

		image := "registry.redhat.io/openshift-service-mesh/istio-must-gather-rhel8:" + env.Getenv("MUSTGATHERTAG", "2.3")

		t.LogStepf("Capture must-gather using image %s", image)
		dir := shell.CreateTempDir(t, "must-gather-")
		shell.Executef(t, `mkdir -p %s; oc adm must-gather --dest-dir=%s --image=%s`, dir, dir, image)

		t.LogStep("Check cluster-scoped openshift-operators.servicemesh-resources.maistra.io.yaml")
		pattern := dir + "/*must-gather*/cluster-scoped-resources/admissionregistration.k8s.io/mutatingwebhookconfigurations/openshift-operators.servicemesh-resources.maistra.io.yaml"
		matches, err := filepath.Glob(pattern)
		if err == nil && len(matches) != 0 {
			t.LogSuccessf("file exists: %s", matches)
		} else {
			t.Fatalf("openshift-operators.servicemesh-resources.maistra.io.yaml file not found: %s", matches)
		}
	})
}
