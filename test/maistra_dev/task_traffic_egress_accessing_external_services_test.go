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

func cleanupAccessExternalServices(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(testNamespace, httpbinextTimeout, kubeconfig)
	util.KubeDeleteContents(namespace, googleextServiceEntry, kubeconfig)
	util.KubeDeleteContents(namespace, httbinextServiceEntry, kubeconfig)
	util.Shell("kubectl get configmap istio -n %s -o yaml | sed 's/mode: REGISTRY_ONLY/mode: ALLOW_ANY/g' | kubectl replace -n %s -f -", meshNamespace, meshNamespace)
	cleanSleep(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestAccessExternalServices(t *testing.T) {
	defer cleanupAccessExternalServices(testNamespace)
	defer recoverPanic(t)

	log.Info("# TestAccessExternalServices")
	deploySleep(testNamespace)
	sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfig)
	util.Inspect(err, "Failed to get sleep pod name", "", t)

	t.Run("TrafficManagement_egress_envoy_passthrough_to_external_services", func(t *testing.T) {
		defer recoverPanic(t)
		log.Info("check the sidecar outboundTrafficPolicy mode")
		util.Shell("kubectl get configmap istio -n %s -o yaml | grep -o \"mode: ALLOW_ANY\"", meshNamespace)

		log.Info("make requests to external https services")
		command := "curl -I https://www.google.com | grep  \"HTTP/\""
		msg, err := util.PodExec(testNamespace, sleepPod, "sleep", command, false, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "200") {
			log.Infof("Success. Get https://www.google.com response: %s", msg)
		} else {
			log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}
	})

	t.Run("TrafficManagement_egress_controlled_access_to_external_httpbin_services", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("update global.outboundTrafficPolicy.mode")
		util.Shell("kubectl get configmap istio -n %s -o yaml | sed 's/mode: ALLOW_ANY/mode: REGISTRY_ONLY/g' | kubectl replace -n %s -f -", meshNamespace, meshNamespace)
		time.Sleep(time.Duration(waitTime*2) * time.Second)
		command := "curl -I https://www.google.com | grep  \"HTTP/\""
		_, err := util.PodExec(testNamespace, sleepPod, "sleep", command, false, kubeconfig)
		if err != nil {
			log.Infof("Expected requests to external services are blocked")
		} else {
			time.Sleep(time.Duration(waitTime*2) * time.Second)
		}

		log.Info("create a ServiceEntry to external httpbin")
		util.KubeApplyContents(testNamespace, httbinextServiceEntry, kubeconfig)
		time.Sleep(time.Duration(waitTime) * time.Second)
		command = "curl http://httpbin.org/headers"
		msg, err := util.PodExec(testNamespace, sleepPod, "sleep", command, false, kubeconfig)
		if err != nil {
			log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		} else {
			log.Infof("Success. Get http://httpbin.org/headers response:\n%s", msg)
		}
	})

	t.Run("TrafficManagement_egress_access_to_external_https_google", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("create a ServiceEntry to external https google.com")
		util.KubeApplyContents(testNamespace, googleextServiceEntry, kubeconfig)
		time.Sleep(time.Duration(waitTime) * time.Second)
		command := "curl -I https://www.google.com | grep  \"HTTP/\""
		msg, err := util.PodExec(testNamespace, sleepPod, "sleep", command, false, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "200") {
			log.Infof("Success. Get https://www.google.com response: %s", msg)
		} else {
			log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}
	})

	t.Run("TrafficManagement_egress_manage_traffic_to_external_services", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("create a httpbin-ext timeout")
		util.KubeApplyContents(testNamespace, httpbinextTimeout, kubeconfig)
		time.Sleep(time.Duration(waitTime) * time.Second)
		command := "sh -c \"curl -o /dev/null -s -w '%{http_code}' http://httpbin.org/delay/5\""
		msg, err := util.PodExec(testNamespace, sleepPod, "sleep", command, false, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if msg == "504" {
			log.Infof("Get expected response failure: %s", msg)
		} else {
			log.Infof("Error response code: %s", msg)
			t.Errorf("Error response code: %s", msg)
		}
	})
}