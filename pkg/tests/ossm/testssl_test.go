// Copyright 2021 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestSSL(t *testing.T) {
	NewTest(t).Id("T27").Groups(Full, InterOp, ARM, Patching).Run(func(t TestHelper) {
		t.Cleanup(func() {
			oc.Patch(t, meshNamespace, "smcp", smcpName, "json", `[{"op": "remove", "path": "/spec/security/controlPlane/tls"}]`)
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
spec:
  security:
    dataPlane:
      mtls: false
    controlPlane:
      mtls: false
`)
			app.Uninstall(t, app.BookinfoWithMTLS(ns.Bookinfo))
			oc.DeleteFromTemplate(t, ns.Bookinfo, testSSLDeployment, nil)
		})

		DeployControlPlane(t) // TODO: integrate below patch here

		t.LogStep("Patch SMCP to enable mTLS in dataPlane and controlPlane and set min/maxProtocolVersion, cipherSuites, and ecdhCurves")
		oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
spec:
  security:
    dataPlane:
      mtls: true
    controlPlane:
      mtls: true
      tls:
        minProtocolVersion: TLSv1_2
        maxProtocolVersion: TLSv1_2
        cipherSuites:
        - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
        ecdhCurves:
        - CurveP256
        - CurveP384
`)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Install bookinfo with mTLS and testssl pod")
		oc.ApplyTemplate(t, ns.Bookinfo, testSSLDeployment, nil)
		app.InstallAndWaitReady(t, app.BookinfoWithMTLS(ns.Bookinfo))
		oc.WaitDeploymentRolloutComplete(t, ns.Bookinfo, "testssl")

		t.LogStep("Check testssl.sh results")
		command := "testssl.sh -P -6 --ssl-native productpage:9080 || true"

		retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(10), func(t TestHelper) {
			oc.Exec(t,
				pod.MatchingSelector("app=testssl", ns.Bookinfo),
				"testssl",
				command,
				assert.OutputContains(
					"TLSv1.2",
					"Received the TLSv1.2 needed in the testssl.sh results",
					"Expected to receive TLSv1.2 in the testssl.sh results, but received something different"),
				assert.OutputContains(
					"ECDHE-RSA-AES128-GCM-SHA256",
					"Results received the correct SHA256",
					"Results not include: ECDHE-RSA-AES128-GCM-SHA256"),
				assert.OutputContains(
					"prime256v1",
					"Results included: prime256v1",
					"Results not include:prime256v1"))
		})
	})
}

const testSSLDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: testssl
spec:
  replicas: 1
  selector:
    matchLabels:
      app: testssl
  template:
    metadata:
      labels:
        app: testssl
    spec:
      terminationGracePeriodSeconds: 0
      containers:
      - name: testssl
        image: {{ image "testssl" }}
        command: ["tail", "-f", "/dev/null"]
`
