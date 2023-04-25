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

package authorization

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestTrustDomainMigration(t *testing.T) {
	NewTest(t).Id("T24").Groups(Full, InterOp).Run(func(t TestHelper) {
		foo := "foo"
		bar := "bar"

		t.Cleanup(func() {
			oc.DeleteFromString(t, foo, TrustDomainPolicy)
			oc.Patch(t, meshNamespace, "smcp", smcpName, "json", `[{"op": "remove", "path": "/spec/security"}]`)

			app.Uninstall(t,
				app.Httpbin(foo),
				app.Sleep(foo),
				app.Sleep(bar))
		})

		ossm.DeployControlPlane(t) // integrate the applyTrustDomain patch here

		applyTrustDomain(t, "old-td", "", true)

		app.InstallAndWaitReady(t,
			app.Httpbin(foo),
			app.Sleep(foo),
			app.Sleep(bar))

		t.Log("Apply deny all policy except sleep in bar namespace")
		oc.ApplyString(t, foo, TrustDomainPolicy)

		t.NewSubTest("Case 1: Verifying policy works").Run(func(t TestHelper) {
			runCurlInSleepPod(t, foo, http.StatusForbidden)
			runCurlInSleepPod(t, bar, http.StatusOK)
		})

		t.NewSubTest("Case 2: Migrate trust domain without trust domain aliases").Run(func(t TestHelper) {
			applyTrustDomain(t, "new-td", "", true)
			oc.RestartAllPodsAndWaitReady(t, foo, bar)

			runCurlInSleepPod(t, foo, http.StatusForbidden)
			runCurlInSleepPod(t, bar, http.StatusForbidden)
		})

		t.NewSubTest("Case 3: Migrate trust domain with trust domain aliases").Run(func(t TestHelper) {
			applyTrustDomain(t, "new-td", "old-td", true)
			oc.RestartAllPodsAndWaitReady(t, foo, bar)

			runCurlInSleepPod(t, foo, http.StatusForbidden)
			runCurlInSleepPod(t, bar, http.StatusOK)
		})
	})
}

func runCurlInSleepPod(t TestHelper, ns string, expectedStatus int) {
	t.Logf("Verifying curl output, expecting %d", expectedStatus)
	retry.UntilSuccess(t, func(t TestHelper) {
		oc.Exec(t,
			pod.MatchingSelector("app=sleep", ns),
			"sleep",
			`curl http://httpbin.foo:8000/ip -sS -o /dev/null -w "%{http_code}\n"`,
			assert.OutputContains(strconv.Itoa(expectedStatus), "", ""))
	})
}

func applyTrustDomain(t TestHelper, domain, alias string, mtls bool) {
	t.Logf("Configure spec.security.trust.domain to %q and alias %q", domain, alias)

	if alias != "" {
		alias = fmt.Sprintf("%q", alias)
	}

	oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", fmt.Sprintf(`
spec:
  security:
    dataPlane:
      mtls: %v
    trust:
      domain: %s
      additionalDomains: [%s]
`, mtls, domain, alias))

	oc.WaitSMCPReady(t, meshNamespace, smcpName)
	if env.GetSMCPVersion().LessThan(version.SMCP_2_2) {
		t.Log("Restarting deployments")
		oc.RestartAllPods(t, meshNamespace)
	}
}

const TrustDomainPolicy = `
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: service-httpbin.foo.svc.cluster.local
spec:
  rules:
  - from:
    - source:
        principals:
        - old-td/ns/bar/sa/sleep
    to:
    - operation:
        methods:
        - GET
  selector:
    matchLabels:
      app: httpbin
`
