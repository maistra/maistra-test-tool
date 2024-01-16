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
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/require"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/template"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestRequestRouting(t *testing.T) {
	NewTest(t).Id("T1").Groups(Smoke, Full, InterOp, ARM).Run(func(t TestHelper) {
		ns := "bookinfo"

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
			os.Remove(`../../../../testdata/resources/html/modified-productpage-test-user-v2.html`)
			os.Remove(`../../../../testdata/resources/html/modified-productpage-normal-user-v1.html`)
		})

		ossm.DeployControlPlane(t)

		t.LogStep("Install Bookinfo")
		app.InstallAndWaitReady(t, app.Bookinfo(ns))

		productpageURL := app.BookinfoProductPageURL(t, meshNamespace)
		testUserCookieJar := app.BookinfoLogin(t, meshNamespace)

		t.NewSubTest("not-logged-in").Run(func(t TestHelper) {
			oc.ApplyString(t, ns, app.BookinfoVirtualServicesAllV1)

			reviewV1Podname := strings.TrimSpace(shell.Execute(t, fmt.Sprintf(`oc get pods -n %s | grep reviews-v1 | awk '{print $1}'`, ns)))
			templateString, err := os.ReadFile("../../../../testdata/resources/html/productpage-normal-user-v1.html")
			if err != nil {
				t.Fatalf("could not read template file %s: %v", templateString, err)
			}
			htmlFile := template.Run(t, string(templateString), struct{ ReviewV1Podname string }{ReviewV1Podname: reviewV1Podname})
			fmt.Println(htmlFile)
			os.WriteFile("../../../../testdata/resources/html/modified-productpage-normal-user-v1.html", []byte(htmlFile), 0644)

			t.LogStep("get productpage without logging in; expect to get reviews-v1 (5x)")
			retry.UntilSuccess(t, func(t TestHelper) {
				for i := 0; i < 5; i++ {
					curl.Request(t,
						productpageURL, nil,
						require.ResponseMatchesFile(
							"modified-productpage-normal-user-v1.html",
							"productpage called reviews-v1",
							"expected productpage to call reviews-v1, but got an unexpected response",
							app.ProductPageResponseFiles...))
				}
			})
		})

		t.NewSubTest("logged-in").Run(func(t TestHelper) {
			oc.ApplyString(t, ns, app.BookinfoVirtualServiceReviewsV2)

			reviewV2Podname := strings.TrimSpace(shell.Execute(t, fmt.Sprintf(`oc get pods -n %s | grep reviews-v2 | awk '{print $1}'`, ns)))
			template2String, err := os.ReadFile("../../../../testdata/resources/html/productpage-test-user-v2.html")
			if err != nil {
				t.Fatalf("could not read template file %s: %v", template2String, err)
			}
			html2File := template.Run(t, string(template2String), struct{ ReviewV2Podname string }{ReviewV2Podname: reviewV2Podname})
			os.WriteFile("../../../../testdata/resources/html/modified-productpage-test-user-v2.html", []byte(html2File), 0644)

			t.LogStep("get productpage as logged-in user; expect to get reviews-v2 (5x)")
			retry.UntilSuccess(t, func(t TestHelper) {
				for i := 0; i < 5; i++ {
					curl.Request(t,
						productpageURL,
						curl.WithCookieJar(testUserCookieJar),
						require.ResponseMatchesFile(
							"modified-productpage-test-user-v2.html",
							"productpage called reviews-v2",
							"expected productpage to call reviews-v2, but got an unexpected response",
							app.ProductPageResponseFiles...))
				}
			})
		})
	})
}
