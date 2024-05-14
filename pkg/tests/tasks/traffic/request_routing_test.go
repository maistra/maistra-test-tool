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
	"os"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/require"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestRequestRouting(t *testing.T) {
	test.NewTest(t).Id("T1").Groups(test.Smoke, test.Full, test.InterOp, test.ARM).Run(func(t test.TestHelper) {

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns.Bookinfo)
			os.Remove(env.GetRootDir() + `/testdata/resources/html/modified-productpage-test-user-v2.html`)
			os.Remove(env.GetRootDir() + `/testdata/resources/html/modified-productpage-normal-user-v1.html`)
		})

		ossm.DeployControlPlane(t)

		t.LogStep("Install Bookinfo")
		app.InstallAndWaitReady(t, app.Bookinfo(ns.Bookinfo))

		productpageURL := app.BookinfoProductPageURL(t, meshNamespace)
		testUserCookieJar := app.BookinfoLogin(t, meshNamespace)

		t.NewSubTest("not-logged-in").Run(func(t test.TestHelper) {
			oc.ApplyString(t, ns.Bookinfo, app.BookinfoVirtualServicesAllV1)

			expectedResponseFile := TestreviewV1(t, "productpage-normal-user-v1.html")

			t.LogStep("get productpage without logging in; expect to get reviews-v1 (5x)")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				for i := 0; i < 5; i++ {
					curl.Request(t,
						productpageURL, nil,
						require.ResponseMatchesFile(
							expectedResponseFile,
							"productpage called reviews-v1",
							"expected productpage to call reviews-v1, but got an unexpected response",
							app.ProductPageResponseFiles...))
				}
			})
		})

		t.NewSubTest("logged-in").Run(func(t test.TestHelper) {
			oc.ApplyString(t, ns.Bookinfo, app.BookinfoVirtualServiceReviewsV2)

			expectedResponseFile2 := TestreviewV2(t, "productpage-test-user-v2.html")

			t.LogStep("get productpage as logged-in user; expect to get reviews-v2 (5x)")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				for i := 0; i < 5; i++ {
					curl.Request(t,
						productpageURL,
						curl.WithCookieJar(testUserCookieJar),
						require.ResponseMatchesFile(
							expectedResponseFile2,
							"productpage called reviews-v2",
							"expected productpage to call reviews-v2, but got an unexpected response",
							app.ProductPageResponseFiles...))
				}
			})
		})
	})
}
