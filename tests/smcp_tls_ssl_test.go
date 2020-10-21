// Copyright 2020 Red Hat, Inc.
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

package tests

import (
	"maistra/util"
	"strings"
	"testing"
	"time"

	"istio.io/pkg/log"
)

func cleanupTestSSL(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(namespace, testSSLDeployment, kubeconfig)
	cleanBookinfo(namespace)

	util.Shell(`kubectl patch -n %s smcp/%s --type=json -p='[{"op": "remove", "path": "/spec/security/controlPlane/tls"}]'`, meshNamespace, smcpName)
	util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":false},"controlPlane":{"mtls":false}}}}'`, meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*8) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=ingressgateway", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=egressgateway", kubeconfig)
}

func TestSSL(t *testing.T) {
	defer cleanupTestSSL(testNamespace)

	t.Run("Operator_test_smcp_testssl", func(t *testing.T) {

		defer recoverPanic(t)

		// update mtls to true
		log.Info("Update SMCP mtls to true")
		util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":true},"controlPlane":{"mtls":true}}}}'`, meshNamespace, smcpName)
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		log.Info("Update SMCP spec.security.controlPlane.tls")

		util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{%s:{%s,%s,%s,%s}}}}}'`,
			meshNamespace, smcpName,
			`"spec":{"security":{"controlPlane":{"tls"`,
			`"minProtocolVersion":"TLSv1_2"`,
			`"maxProtocolVersion":"TLSv1_2"`,
			`"cipherSuites":["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"]`,
			`"ecdhCurves":["CurveP256", "CurveP384"]`)

		time.Sleep(time.Duration(waitTime*8) * time.Second)
		util.CheckPodRunning(meshNamespace, "istio=ingressgateway", kubeconfig)
		util.CheckPodRunning(meshNamespace, "istio=egressgateway", kubeconfig)

		log.Info("Deploy bookinfo")
		deployBookinfo(testNamespace, true)

		log.Info("Deploy testssl pod")
		util.KubeApplyContents(testNamespace, testSSLDeployment, kubeconfig)
		util.CheckPodRunning(testNamespace, "app=testssl", kubeconfig)

		log.Info("Check testssl.sh results. Ignore info	Command error")
		pod, err := util.GetPodName(testNamespace, "app=testssl", kubeconfig)
		util.Inspect(err, "failed to get testssl pod", "", t)

		command := "./testssl/testssl.sh productpage:9080"
		msg, err := util.PodExec(testNamespace, pod, "testssl", command, false, kubeconfig)
		if !strings.Contains(msg, "TLSv1.2") {
			t.Errorf("Results not include: TLSv1.2")
			log.Errorf("Results not include: TLSv1.2")
		}
		if !strings.Contains(msg, "ECDHE-RSA-AES128-GCM-SHA256") {
			t.Errorf("Results not include: ECDHE-RSA-AES128-GCM-SHA256")
			log.Errorf("Results not include: ECDHE-RSA-AES128-GCM-SHA256")
		}
		if !strings.Contains(msg, "P-256") {
			t.Errorf("Results not include: P-256")
			log.Errorf("Results not include: P-256")
		}

	})
}
