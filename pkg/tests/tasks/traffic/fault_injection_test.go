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

package traffic

import (
	_ "embed"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/virtualservice-ratings-fixed-delay.yaml
	ratingsVirtualServiceWithFixedDelay string

	//go:embed yaml/virtualservice-ratings-abort500.yaml
	ratingsVirtualServiceWithHttpStatus500 string
)

func TestFaultInjection(t *testing.T) {
	NewTest(t).Id("T2").Groups(Full, InterOp, ARM).Run(func(t TestHelper) {
		ns := "bookinfo"

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		app.InstallAndWaitReady(t, app.Bookinfo(ns))

		testUserCookieJar := app.BookinfoLogin(t, meshNamespace)

		oc.ApplyString(t, ns, app.BookinfoVirtualServicesAllV1)
		oc.ApplyString(t, ns, app.BookinfoVirtualServiceReviewsV2)

		t.NewSubTest("ratings-fault-delay").Run(func(t TestHelper) {
			oc.ApplyString(t, ns, ratingsVirtualServiceWithFixedDelay)

			t.LogStep("check if productpage shows 'error fetching product reviews' due to delay injection")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					app.BookinfoProductPageURL(t, meshNamespace),
					curl.WithCookieJar(testUserCookieJar),
					assert.DurationInRange(4*time.Second, 14*time.Second),
					assert.ResponseMatchesFile(
						"productpage-test-user-v2-review-timeout.html",
						"productpage shows 'error fetching product reviews', which is expected",
						"expected productpage to show 'error fetching product reviews', but got a different response",
						app.ProductPageResponseFiles...))
			})
		})

		t.NewSubTest("ratings-fault-abort").Run(func(t TestHelper) {
			oc.ApplyString(t, ns, ratingsVirtualServiceWithHttpStatus500)

			t.LogStep("check if productpage shows ratings service as unavailable due to abort injection")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					app.BookinfoProductPageURL(t, meshNamespace),
					curl.WithCookieJar(testUserCookieJar),
					assert.ResponseMatchesFile(
						"productpage-test-user-v2-rating-unavailable.html",
						"productpage shows 'ratings service is currently unavailable' as expected",
						"expected productpage to show ratings service as unavailable, but got a different response",
						app.ProductPageResponseFiles...))
			})
		})
	})
}
