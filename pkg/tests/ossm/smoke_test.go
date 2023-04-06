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
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestBookinfoInjection(t *testing.T) {
	NewTest(t).Id("A2").Groups(ARM, Full, Smoke, InterOp).Run(func(t TestHelper) {

		ns := "bookinfo"

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		app.InstallAndWaitReady(t, app.Bookinfo(ns))

		t.LogStep("Check pods running 2/2 ready and with Sidecar Injection")
		assertSidecarInjectedInAllPods(t, ns)

		t.LogStep("Check if bookinfo productpage is running through the Proxy")
		shell.Execute(t,
			fmt.Sprintf(`oc -n %s run -it --restart=Never --rm curl --image curlimages/curl -- curl -I http://productpage:9080`, ns),
			assert.OutputContains(
				"HTTP/1.1 200 OK",
				"server: istio-envoy",
				"x-envoy-decorator-operation: productpage.bookinfo.svc.cluster.local:9080"))
	})
}

func assertSidecarInjectedInAllPods(t TestHelper, ns string) {
	response := util.GetPodNames(ns)
	t.Log(response)
	for _, podName := range response {
		shell.Execute(t,
			fmt.Sprintf(`oc get pod %s -n %s`, podName, ns),
			assert.OutputContains(
				"2/2",
				"Proxy container is injected and running",
				"Proxy container is not running"))
	}
}
