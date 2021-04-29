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
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupEgressGateways(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(namespace, cnnextGatewayHTTPS, kubeconfig)
	cleanSleep(namespace)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
}

func TestEgressGateways(t *testing.T) {
	defer cleanupEgressGateways(testNamespace)
	defer recoverPanic(t)

	log.Info("# TestEgressGateways")

	deploySleep(testNamespace)
	sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfig)
	util.Inspect(err, "Failed to get sleep pod name", "", t)

	t.Run("TrafficManagement_egress_gateway_for_http_traffic", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("create a Gateway to external edition.cnn.com")
		util.KubeApplyContents(testNamespace, cnnextGateway, kubeconfig)
		// OCP Route created by ior
		time.Sleep(time.Duration(waitTime*4) * time.Second)
		command := "curl -sL -o /dev/null -D - http://edition.cnn.com/politics"
		msg, err := util.PodExec(testNamespace, sleepPod, "sleep", command, false, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "301 Moved Permanently") {
			log.Infof("Success. Get http://edition.cnn.com/politics response: %s", msg)
		} else {
			log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}

		util.KubeDeleteContents(testNamespace, cnnextGateway, kubeconfig)
		time.Sleep(time.Duration(waitTime*2) * time.Second)
	})

	t.Run("TrafficManagement_egress_gateway_for_https_traffic", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("create a https Gateway to external edition.cnn.com")
		util.KubeApplyContents(testNamespace, cnnextGatewayHTTPS, kubeconfig)
		// OCP Route created by ior
		time.Sleep(time.Duration(waitTime*2) * time.Second)
		command := "curl -sL -o /dev/null -D - https://edition.cnn.com/politics"
		msg, err := util.PodExec(testNamespace, sleepPod, "sleep", command, false, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "HTTP/2 200") {
			log.Infof("Success. Get https://edition.cnn.com/politics response: %s", msg)
		} else {
			log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}
	})

	// Apply Kubernetes network policies

}
