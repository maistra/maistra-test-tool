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

package ossm

import (
	"os"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupTestSSL() {
	util.Log.Info("Cleanup ...")
	bookinfo := examples.Bookinfo{"bookinfo"}
	util.KubeDeleteContents("bookinfo", testSSLDeployment)
	bookinfo.Uninstall()

	util.Shell(`kubectl patch -n %s smcp/basic --type=json -p='[{"op": "remove", "path": "/spec/security/controlPlane/tls"}]'`, "istio-system")
	util.Shell(`kubectl patch -n %s smcp/basic --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":false},"controlPlane":{"mtls":false}}}}'`, "istio-system")
	util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 180s`, "istio-system")
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func TestSSL(t *testing.T) {
	defer cleanupTestSSL()

	t.Run("Operator_test_smcp_testssl", func(t *testing.T) {
		defer util.RecoverPanic(t)

		// update mtls to true
		util.Log.Info("Update SMCP mtls to true")
		util.Shell(`kubectl patch -n %s smcp/basic --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":true},"controlPlane":{"mtls":true}}}}'`, "istio-system")
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 180s`, "istio-system")

		util.Log.Info("Update SMCP spec.security.controlPlane.tls")

		util.Shell(`kubectl patch -n %s smcp/basic --type merge -p '{%s:{%s,%s,%s,%s}}}}}'`,
			"istio-system",
			`"spec":{"security":{"controlPlane":{"tls"`,
			`"minProtocolVersion":"TLSv1_2"`,
			`"maxProtocolVersion":"TLSv1_2"`,
			`"cipherSuites":["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"]`,
			`"ecdhCurves":["CurveP256", "CurveP384"]`)

		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 180s`, "istio-system")

		util.Log.Info("Deploy bookinfo")
		bookinfo := examples.Bookinfo{"bookinfo"}
		bookinfo.Install(true)

		util.Log.Info("Deploy testssl pod")
		if getenv("SAMPLEARCH", "x86") == "p" {
			util.KubeApplyContents("bookinfo", testSSLDeploymentP)
		} else if getenv("SAMPLEARCH", "x86") == "z" {
			util.KubeApplyContents("bookinfo", testSSLDeploymentZ)
		} else {
			util.KubeApplyContents("bookinfo", testSSLDeployment)
		}
		util.CheckPodRunning("bookinfo", "app=testssl")

		util.Log.Info("Check testssl.sh results. Ignore info	Command error")
		pod, err := util.GetPodName("bookinfo", "app=testssl")
		util.Inspect(err, "failed to get testssl pod", "", t)

		command := "./testssl/testssl.sh productpage:9080"
		msg, err := util.PodExec("bookinfo", pod, "testssl", command, false)
		if !strings.Contains(msg, "TLSv1.2") {
			t.Errorf("Results not include: TLSv1.2")
			util.Log.Errorf("Results not include: TLSv1.2")
		}
		if !strings.Contains(msg, "ECDHE-RSA-AES128-GCM-SHA256") {
			t.Errorf("Results not include: ECDHE-RSA-AES128-GCM-SHA256")
			util.Log.Errorf("Results not include: ECDHE-RSA-AES128-GCM-SHA256")
		}
		if !strings.Contains(msg, "P-256") {
			t.Errorf("Results not include: P-256")
			util.Log.Errorf("Results not include: P-256")
		}
	})
}
