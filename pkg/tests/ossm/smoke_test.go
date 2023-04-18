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
	"bufio"
	"fmt"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestBookinfoInjection(t *testing.T) {
	NewTest(t).Id("A2").Groups(ARM, Full, Smoke, InterOp).Run(func(t TestHelper) {
		ns := "bookinfo"

		t.Cleanup(func() {
			app.Uninstall(t, app.Bookinfo(ns), app.SleepNoSidecar(ns))
		})

		DeployControlPlane(t)

		t.LogStep("Install bookinfo pods with sidecar and sleep pod without sidecar")
		app.InstallAndWaitReady(t, app.Bookinfo(ns), app.SleepNoSidecar(ns))

		t.LogStep("Check whether sidecar is injected in all bookinfo pods")
		assertSidecarInjectedInAllBookinfoPods(t, ns)

		t.LogStep("Check if bookinfo productpage is running through the Proxy")
		oc.Exec(t,
			pod.MatchingSelector("app=sleep", ns), "sleep",
			"curl -sI http://productpage:9080",
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

func assertSidecarInjectedInAllBookinfoPods(t TestHelper, ns string) {
	shell.Execute(t,
		fmt.Sprintf(`oc -n %s get pods -l 'app in (productpage,details,reviews,ratings)' --no-headers`, ns),
		func(t TestHelper, input string) {
			scanner := bufio.NewScanner(strings.NewReader(input))
			for scanner.Scan() {
				line := scanner.Text()
				podName := strings.Fields(line)[0]
				if strings.Contains(line, "2/2") {
					t.LogSuccessf("Sidecar injected and running in pod %s", podName)
				} else {
					t.Errorf("Sidecar either not injected or not running in pod %s: %s", podName, line)
				}
			}
		})
}
