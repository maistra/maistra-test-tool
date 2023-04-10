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
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestBookinfoInjection(t *testing.T) {
	NewTest(t).Id("A2").Groups(ARM, Full, Smoke, InterOp).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		ns := "bookinfo"

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		app.InstallAndWaitReady(t, app.Bookinfo(ns))

		t.LogStep("Check pods running 2/2 ready and with Sidecar Injection")
		assertSidecarInjectedInAllPods(t, ns)

		t.LogStep("Check if bookinfo productpage is running through the Proxy")
		shell.Execute(t,
			fmt.Sprintf(`oc -n %s run -i --restart=Never --rm curl --image curlimages/curl -- curl -sI http://productpage:9080`, ns),
			assert.OutputContains(
				"HTTP/1.1 200 OK",
				"ProductPage returns 200 OK",
				"ProductPage didn't return 200 OK"),
			assert.OutputContains(
				"server: istio-envoy",
				"HTTP header 'server: istio-envoy' is present in the response",
				"HTTP header 'server: istio-envoy' is missing from the response"),
			assert.OutputContains(
				"x-envoy-decorator-operation",
				"HTTP header 'x-envoy-decorator-operation' is present in the response",
				"HTTP header 'x-envoy-decorator-operation' is missing from the response"))
	})
}

func assertSidecarInjectedInAllPods(t TestHelper, ns string) {
	response := util.GetPodNames(ns)
	for _, podName := range response {
		shell.Execute(t,
			fmt.Sprintf(`oc get pod %s -n %s`, podName, ns),
			assert.OutputContains(
				"2/2",
				fmt.Sprintf("Proxy container is injected and running in pod %s", podName),
				fmt.Sprintf("Proxy container is not running in pod %s", podName)))
	}
}
