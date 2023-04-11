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
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/smcp-patch-rate-limiting.yaml
	rateLimitSMCPPatch string

	//go:embed yaml/envoyfilter-ratelimit.yaml
	rateLimitFilterYaml_template string
)

func TestRateLimiting(t *testing.T) {
	NewTest(t).Id("T28").Groups(Full).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		namespaces := []string{"bookinfo", "redis"}
		t.Cleanup(func() {
			oc.RecreateNamespace(t, meshNamespace)
			oc.DeleteNamespace(t, namespaces...)
		})
		t.LogStep("Install Bookinfo and Redis")
		app.InstallAndWaitReady(t, app.Bookinfo(namespaces[0]), app.Redis(namespaces[1]))

		t.LogStep("Patch SMCP to enable rate limiting and wait until smcp is ready")
		oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", rateLimitSMCPPatch)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Verify rls Pod is Running")
		rlsPod := pod.MatchingSelector("app=rls", meshNamespace)
		oc.WaitPodRunning(t, rlsPod)

		t.LogStep("Create EnvoyFilter for rate limiting")
		oc.ApplyTemplate(t, meshNamespace, rateLimitFilterYaml_template, Smcp)

		productPageURL := app.BookinfoProductPageURL(t, meshNamespace)
		t.LogStep("Make 3 request to validate rate limit: first should work, second should fail with 429, third should work again after wait more than 10 seconds")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			curl.Request(t, productPageURL, nil, assert.ResponseStatus(200))
			curl.Request(t, productPageURL, nil, assert.ResponseStatus(429))
			time.Sleep(time.Second * 5)
			curl.Request(t, productPageURL, nil, assert.ResponseStatus(200))
		})
	})
}
