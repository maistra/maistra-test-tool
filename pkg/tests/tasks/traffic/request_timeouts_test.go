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
	"os"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/check/require"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

//go:embed yaml/virtualservice-reviews-ratings-timeout.yaml
var reviewTimeout string

func TestRequestTimeouts(t *testing.T) {
	NewTest(t).Id("T5").Groups(Full, InterOp, ARM, Disconnected, Persistent).Run(func(t TestHelper) {

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns.Bookinfo)
			os.Remove(env.GetRootDir() + `/testdata/resources/html/modified-productpage-normal-user-v1.html`)
		})

		ossm.DeployControlPlane(t)

		t.LogStep("Install Bookinfo")
		app.InstallAndWaitReady(t, app.Bookinfo(ns.Bookinfo))

		productpageURL := app.BookinfoProductPageURL(t, meshNamespace)

		oc.ApplyString(t, ns.Bookinfo, app.BookinfoVirtualServicesAllV1)

		t.LogStep("make sure there is no timeout before applying delay and timeout in VirtualServices")

		expectedResponseFile := TestreviewV1(t, "productpage-normal-user-v1.html")

		retry.UntilSuccess(t, func(t TestHelper) {
			curl.Request(t,
				productpageURL, nil,
				assert.ResponseMatchesFile(
					expectedResponseFile,
					"received normal productpage response",
					"unexpected response",
					app.ProductPageResponseFiles...))
		})

		t.LogStep("apply delay and timeout in VirtualServices")
		oc.ApplyString(t, ns.Bookinfo, reviewTimeout)

		t.LogStep("check if productpage shows 'error fetching product reviews' due to delay and timeout injection")
		retry.UntilSuccess(t, func(t TestHelper) {
			for i := 0; i <= 5; i++ {
				curl.Request(t,
					productpageURL, nil,
					require.ResponseMatchesFile(
						"productpage-review-timeout.html",
						"productpage shows 'error fetching product reviews', which is expected",
						"expected productpage to show 'error fetching product reviews', but got a different response",
						app.ProductPageResponseFiles...))
			}
		})
	})
}
