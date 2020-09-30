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

func cleanupTestTLSVersionSMCP() {
	log.Info("# Cleanup ...")
	util.Shell("kubectl patch -n %s smcp/%s --type=json -p='[{\"op\": \"remove\", \"path\": \"/spec/security/controlPlane/tls\"}]'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*8) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=ingressgateway", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=egressgateway", kubeconfig)
}

func TestTLSVersionSMCP(t *testing.T) {
	defer cleanupTestTLSVersionSMCP()

	t.Run("Operator_test_smcp_global_tls_minVersion_TLSv1_0", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Update SMCP spec.security.controlPlane.tls.minProtocolVersion: TLSv1_0")
		util.Shell("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"security\":{\"controlPlane\":{\"tls\":{\"minProtocolVersion\":\"TLSv1_0\"}}}}}'", meshNamespace, smcpName)
		time.Sleep(time.Duration(waitTime*8) * time.Second)
		util.CheckPodRunning(meshNamespace, "istio=ingressgateway", kubeconfig)
		util.CheckPodRunning(meshNamespace, "istio=egressgateway", kubeconfig)

		msg, _ := util.Shell("kubectl get -n %s smcp/%s", meshNamespace, smcpName)
		if strings.Contains(msg, "ComponentsReady") {
			log.Info(msg)
		} else {
			t.Errorf("Failed to update SMCP with spec.security.controlPlane.tls.minProtocolVersion: TLSv1_0")
		}
	})

	t.Run("Operator_test_smcp_global_tls_minVersion_TLSv1_1", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Update SMCP spec.security.controlPlane.tls.minProtocolVersion: TLSv1_1")
		util.Shell("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"security\":{\"controlPlane\":{\"tls\":{\"minProtocolVersion\":\"TLSv1_1\"}}}}}'", meshNamespace, smcpName)
		time.Sleep(time.Duration(waitTime*8) * time.Second)
		util.CheckPodRunning(meshNamespace, "istio=ingressgateway", kubeconfig)
		util.CheckPodRunning(meshNamespace, "istio=egressgateway", kubeconfig)

		msg, _ := util.Shell("kubectl get -n %s smcp/%s", meshNamespace, smcpName)
		if strings.Contains(msg, "ComponentsReady") {
			log.Info(msg)
		} else {
			t.Errorf("Failed to update SMCP with spec.security.controlPlane.tls.minProtocolVersion: TLSv1_1")
		}
	})

	t.Run("Operator_test_smcp_global_tls_maxVersion_TLSv1_3", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Update SMCP spec.security.controlPlane.tls.minProtocolVersion: TLSv1_3")
		util.Shell("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"security\":{\"controlPlane\":{\"tls\":{\"maxProtocolVersion\":\"TLSv1_3\"}}}}}'", meshNamespace, smcpName)
		time.Sleep(time.Duration(waitTime*8) * time.Second)
		util.CheckPodRunning(meshNamespace, "istio=ingressgateway", kubeconfig)
		util.CheckPodRunning(meshNamespace, "istio=egressgateway", kubeconfig)

		msg, _ := util.Shell("kubectl get -n %s smcp/%s", meshNamespace, smcpName)
		if strings.Contains(msg, "ComponentsReady") {
			log.Info(msg)
		} else {
			t.Errorf("Failed to update SMCP with spec.security.controlPlane.tls.maxProtocolVersion: TLSv1_3")
		}
	})
}
