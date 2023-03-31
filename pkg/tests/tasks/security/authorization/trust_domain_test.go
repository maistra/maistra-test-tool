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

package authorizaton

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestTrustDomainMigration(t *testing.T) {
	NewTest(t).Id("T24").Groups(Full, InterOp).Run(func(t TestHelper) {
		fooNamespace := "foo"
		barNamespace := "bar"

		defer func() {
			t.Log("Cleanup")
			oc.DeleteFromString(t, fooNamespace, TrustDomainPolicy)
			app.Uninstall(t,
				app.Httpbin(fooNamespace),
				app.Sleep(fooNamespace),
				app.Sleep(barNamespace))
			applyTrustDomain(t, "cluster.local", "", false)
		}()

		t.Log("Trust Domain Migration")
		applyTrustDomain(t, "old-td", "", true)

		// Deploy workloads
		app.InstallAndWaitReady(t,
			app.Httpbin(fooNamespace),
			app.Sleep(fooNamespace),
			app.Sleep(barNamespace))

		t.Log("Apply deny all policy except sleep in bar namespace")
		oc.ApplyString(t, fooNamespace, TrustDomainPolicy)

		t.NewSubTest("Case 1: Verifying policy works").Run(func(t TestHelper) {
			runCurlInSleepPod(t, fooNamespace, http.StatusForbidden)
			runCurlInSleepPod(t, barNamespace, http.StatusOK)
		})

		t.NewSubTest("Case 2: Migrate trust domain without trust domain aliases").Run(func(t TestHelper) {
			applyTrustDomain(t, "new-td", "", true)
			oc.RestartAllPodsAndWaitReady(t, fooNamespace, barNamespace)

			runCurlInSleepPod(t, fooNamespace, http.StatusForbidden)
			runCurlInSleepPod(t, barNamespace, http.StatusForbidden)
		})

		t.NewSubTest("Case 3: Migrate trust domain with trust domain aliases").Run(func(t TestHelper) {
			applyTrustDomain(t, "new-td", "old-td", true)
			oc.RestartAllPodsAndWaitReady(t, fooNamespace, barNamespace)

			runCurlInSleepPod(t, fooNamespace, http.StatusForbidden)
			runCurlInSleepPod(t, barNamespace, http.StatusOK)
		})
	})
}

func runCurlInSleepPod(t TestHelper, ns string, expectedStatus int) {
	t.Logf("Verifying curl output, expecting %d", expectedStatus)
	retry.UntilSuccess(t, func(t TestHelper) {
		oc.Exec(t,
			pod.MatchingSelector("app=sleep", ns),
			"sleep",
			`curl http://httpbin.foo:8000/ip -sS -o /dev/null -w "%%{http_code}\n"`,
			assert.OutputContains(strconv.Itoa(expectedStatus), "", ""))
	})
}

func applyTrustDomain(t TestHelper, domain, alias string, mtls bool) {
	t.Logf("Configuring  spec.security.trust.domain to %q and alias %q", domain, alias)

	if alias != "" {
		alias = fmt.Sprintf("%q", alias)
	}

	shell.Executef(t, `oc -n %s patch smcp/%s --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":%v}, "trust":{"domain":"%s", "additionalDomains": [%s]}}}}'`, meshNamespace, smcpName, mtls, domain, alias)

	// Wait for the operator to reconcile the changes
	oc.WaitSMCPReady(t, meshNamespace, smcpName)

	// TODO: figure out if restarting deployments is necessary; shouldn't the SMCP being ready indicate that the deployments were restarted?
	// Restart istiod so it picks up the new trust domain
	shell.Executef(t, `oc -n %s rollout restart deployment istiod-%s`, meshNamespace, smcpName)

	// Restart ingress gateway since we changed the mtls setting
	shell.Executef(t, `oc -n %s rollout restart deployment istio-ingressgateway`, meshNamespace)

	// wait for both deployments to be restarted (the rollout status command blocks until pods are ready)
	shell.Executef(t, `oc -n %s rollout status deployment istiod-%s`, meshNamespace, smcpName)
	shell.Executef(t, `oc -n %s rollout status deployment istio-ingressgateway`, meshNamespace)
}
