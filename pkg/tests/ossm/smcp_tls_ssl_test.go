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
	_ "embed"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/deployment-testssl-x86.yaml
	testSSLDeployment string

	//go:embed yaml/deployment-testssl-z.yaml
	testSSLDeploymentZ string

	//go:embed yaml/deployment-testssl-p.yaml
	testSSLDeploymentP string
)

func cleanupTestTLSVersionSMCP() {
	log.Log.Info("Cleanup ...")
	util.Shell(`kubectl patch -n %s smcp/%s --type=json -p='[{"op": "remove", "path": "/spec/security/controlPlane/tls"}]'`, meshNamespace, smcpName)
	util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 180s`, meshNamespace, smcpName)
}

func cleanupTestSSL() {
	log.Log.Info("Cleanup ...")
	bookinfo := examples.Bookinfo{"bookinfo"}
	util.KubeDeleteContents("bookinfo", testSSLDeployment)
	bookinfo.Uninstall()

	util.Shell(`kubectl patch -n %s smcp/%s --type=json -p='[{"op": "remove", "path": "/spec/security/controlPlane/tls"}]'`, meshNamespace, smcpName)
	util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":false},"controlPlane":{"mtls":false}}}}'`, meshNamespace, smcpName)
	util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 180s`, meshNamespace, smcpName)
}

func TestTLSVersionSMCP(t *testing.T) {
	test.NewTest(t).Id("T26").Groups(test.Full, test.ARM, test.InterOp).NotRefactoredYet()

	defer cleanupTestTLSVersionSMCP()

	t.Run("Operator_test_smcp_global_tls_minVersion_TLSv1_0", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Update SMCP spec.security.controlPlane.tls.minProtocolVersion: TLSv1_0")
		_, err := util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"controlPlane":{"tls":{"minProtocolVersion":"TLSv1_0"}}}}}'`, meshNamespace, smcpName)
		util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 180s`, meshNamespace, smcpName)
		if err != nil {
			t.Errorf("Failed to update SMCP with tls.maxProtocolVersion: TLSv1_0")
		}
	})

	t.Run("Operator_test_smcp_global_tls_minVersion_TLSv1_1", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Update SMCP spec.security.controlPlane.tls.minProtocolVersion: TLSv1_1")
		_, err := util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"controlPlane":{"tls":{"minProtocolVersion":"TLSv1_1"}}}}}'`, meshNamespace, smcpName)
		util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 180s`, meshNamespace, smcpName)
		if err != nil {
			t.Errorf("Failed to update SMCP with tls.maxProtocolVersion: TLSv1_1")
		}
	})

	t.Run("Operator_test_smcp_global_tls_maxVersion_TLSv1_3", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Update SMCP spec.security.controlPlane.tls.minProtocolVersion: TLSv1_3")
		_, err := util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"controlPlane":{"tls":{"maxProtocolVersion":"TLSv1_3"}}}}}'`, meshNamespace, smcpName)
		util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 180s`, meshNamespace, smcpName)
		if err != nil {
			t.Errorf("Failed to update SMCP with tls.maxProtocolVersion: TLSv1_3")
		}
	})
}

func TestSSL(t *testing.T) {
	test.NewTest(t).Id("T27").Groups(test.Full, test.InterOp).NotRefactoredYet()

	defer cleanupTestSSL()

	t.Run("Operator_test_smcp_testssl", func(t *testing.T) {
		defer util.RecoverPanic(t)

		// update mtls to true
		log.Log.Info("Update SMCP mtls to true")
		util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":true},"controlPlane":{"mtls":true}}}}'`, meshNamespace, smcpName)
		util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 180s`, meshNamespace, smcpName)

		log.Log.Info("Update SMCP spec.security.controlPlane.tls")

		util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{%s:{%s,%s,%s,%s}}}}}'`,
			meshNamespace, smcpName,
			`"spec":{"security":{"controlPlane":{"tls"`,
			`"minProtocolVersion":"TLSv1_2"`,
			`"maxProtocolVersion":"TLSv1_2"`,
			`"cipherSuites":["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"]`,
			`"ecdhCurves":["CurveP256", "CurveP384"]`)

		util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 180s`, meshNamespace, smcpName)

		log.Log.Info("Deploy bookinfo")
		bookinfo := examples.Bookinfo{"bookinfo"}
		bookinfo.Install(true)

		log.Log.Info("Deploy testssl pod")
		if env.Getenv("SAMPLEARCH", "x86") == "p" {
			util.KubeApplyContents("bookinfo", testSSLDeploymentP)
		} else if env.Getenv("SAMPLEARCH", "x86") == "z" {
			util.KubeApplyContents("bookinfo", testSSLDeploymentZ)
		} else {
			util.KubeApplyContents("bookinfo", testSSLDeployment)
		}
		util.CheckPodRunning("bookinfo", "app=testssl")

		log.Log.Info("Check testssl.sh results. Ignore info	Command error")
		pod, err := util.GetPodName("bookinfo", "app=testssl")
		util.Inspect(err, "failed to get testssl pod", "", t)

		command := "./testssl/testssl.sh -6 productpage:9080"
		msg, err := util.PodExec("bookinfo", pod, "testssl", command, false)
		util.Inspect(err, "failed to execute testssl in pod", "", t)
		if !strings.Contains(msg, "TLSv1.2") {
			t.Errorf("Results not include: TLSv1.2")
			log.Log.Errorf("Results not include: TLSv1.2")
		}
		if !strings.Contains(msg, "ECDHE-RSA-AES128-GCM-SHA256") {
			t.Errorf("Results not include: ECDHE-RSA-AES128-GCM-SHA256")
			log.Log.Errorf("Results not include: ECDHE-RSA-AES128-GCM-SHA256")
		}
		if !strings.Contains(msg, "P-256") {
			t.Errorf("Results not include: P-256")
			log.Log.Errorf("Results not include: P-256")
		}
	})
}
