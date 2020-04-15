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

package main

import (
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/istio/pkg/log"
)

func cleanupCitadelHealthCheck(namespace string) {
	log.Info("# Cleanup ...")
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"security\":{\"citadelHealthCheck\":false}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*10) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=citadel", kubeconfig)
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":false,\"mtls\":{\"enabled\":false}}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*10) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=pilot", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=ingressgateway", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=egressgateway", kubeconfig)
}

func TestCitadelHealthCheck(t *testing.T) {
	defer cleanupCitadelHealthCheck(testNamespace)
	defer recoverPanic(t)

	log.Info("Citadel Health Checking")
	// update mtls to true
	log.Info("Update SMCP mtls to true")
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":true,\"mtls\":{\"enabled\":true}}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*10) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=pilot", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=ingressgateway", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=egressgateway", kubeconfig)

	t.Run("Security_citadel_health_checking", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Redeploy Citadel")
		util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"security\":{\"citadelHealthCheck\":true}}}}'", meshNamespace, smcpName)
		time.Sleep(time.Duration(waitTime*10) * time.Second)
		util.CheckPodRunning(meshNamespace, "istio=citadel", kubeconfig)

		log.Info("Verify that health checking works")
		msg, err := util.ShellMuteOutput("kubectl logs `kubectl get po -n %s | grep istio-citadel | awk '{print $1}'` -n %s | grep \"CSR signing service\"", meshNamespace, meshNamespace)
		util.Inspect(err, "Failed to get logs", "", t)
		if !strings.Contains(msg, "CSR") {
			log.Infof("Error no CSR is healthy log")
			t.Errorf("Error no CSR is healthy log")
		} else {
			log.Infof("Success. Get %s", msg)
		}
	})
}
