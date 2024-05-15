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
	_ "embed"
	"fmt"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/version"

	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/smcp-patch-rate-limiting.yaml
	rateLimitSMCPPatch string

	//go:embed yaml/envoyfilter-ratelimit.yaml
	rateLimitFilterTemplate string
)

func TestRateLimiting(t *testing.T) {
	NewTest(t).Id("T28").Groups(Full, ARM).MaxVersion(version.SMCP_2_2).Run(func(t TestHelper) {
		ns := "bookinfo"
		nsRedis := "redis"

		DeployControlPlane(t)

		t.LogStep("Install Bookinfo and Redis")
		app.InstallAndWaitReady(t, app.Bookinfo(ns), app.Redis(nsRedis))
		t.Cleanup(func() {
			app.Uninstall(t, app.Bookinfo(ns), app.Redis(nsRedis))
			oc.DeleteNamespace(t, nsRedis) // namespace redis is only used in this test, so delete it after test
		})

		t.LogStep("Patch SMCP to enable rate limiting and wait until smcp is ready")
		t.Log("Patch configured to allow 1 request per second only")
		oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", rateLimitSMCPPatch)
		t.Cleanup(func() {
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `{"spec":{"techPreview":{"rateLimiting":null}}}`)
		})
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Verify rls Pod is Running")
		shell.Execute(t,
			fmt.Sprintf("oc wait --for=condition=Ready pod -l %s -n %s --timeout=30s", "app=rls", meshNamespace),
			assert.OutputContains("condition met",
				"The rls Pod is running",
				"ERROR: rls pod expected to be running, but it is not"))

		t.LogStep("Create EnvoyFilter for rate limiting")
		oc.ApplyTemplate(t, meshNamespace, rateLimitFilterTemplate, DefaultSMCP())
		t.Cleanup(func() {
			oc.DeleteFromTemplate(t, meshNamespace, rateLimitFilterTemplate, DefaultSMCP())
		})

		productPageURL := app.BookinfoProductPageURL(t, meshNamespace)
		t.LogStep("Make 3 request to validate rate limit: first should work, second should fail with 429, third should work again after wait more than 1 seconds")
		retry.UntilSuccess(t, func(t TestHelper) {
			curl.Request(t, productPageURL, nil, assert.ResponseStatus(200))
			curl.Request(t, productPageURL, nil, assert.ResponseStatus(429))
			time.Sleep(time.Second * 5) // wait 5 seconds to make sure the rate limit is reset
			curl.Request(t, productPageURL, nil, assert.ResponseStatus(200))
		})
	})
}
