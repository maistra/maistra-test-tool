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


func cleanupMirroring(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(namespace, httpbinAllv1, kubeconfig)
	util.KubeDeleteContents(namespace, httpbinService, kubeconfig)
	util.KubeDeleteContents(namespace, httpbinv1, kubeconfig)
	util.KubeDeleteContents(namespace, httpbinv2, kubeconfig)
	util.KubeDeleteContents(namespace, sleepv2, kubeconfig)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}


func TestMirroring(t *testing.T) {
	defer cleanupMirroring(testNamespace)
	defer recoverPanic(t)

	log.Info("# Mirroring")
	if err := util.KubeApplyContents(testNamespace, httpbinv1, kubeconfig); err != nil {
		t.Errorf("Failed to deploy httpbin v1")
		log.Errorf("Failed to deploy httpbin v1")
	}
	if err := util.KubeApplyContents(testNamespace, httpbinv2, kubeconfig); err != nil {
		t.Errorf("Failed to deploy httpbin v2")
		log.Errorf("Failed to deploy httpbin v2")
	}
	util.CheckPodRunning(testNamespace, "app=httpbin,version=v1", kubeconfig)
	util.CheckPodRunning(testNamespace, "app=httpbin,version=v2", kubeconfig)

	if err := util.KubeApplyContents(testNamespace, httpbinService, kubeconfig); err != nil {
		t.Errorf("Failed to apply httpbin service")
		log.Errorf("Failed to apply httpbin service")
	}
	time.Sleep(time.Duration(waitTime*2) * time.Second)

	if err := util.KubeApplyContents(testNamespace, sleepv2, kubeconfig); err != nil {
		t.Errorf("Failed to deploy sleep")
		log.Errorf("Failed to deploy sleep")
	}
	util.CheckPodRunning(testNamespace, "app=sleep", kubeconfig)
	time.Sleep(time.Duration(waitTime*4) * time.Second)

	t.Run("TrafficManagement_creating_a_default_routing_policy", func(t *testing.T) {
		defer recoverPanic(t)

		/*
		If you installed/configured Istio with mutual TLS authentication enabled, you must add a TLS traffic policy mode: ISTIO_MUTUAL to the DestinationRule before applying it. Otherwise requests will generate 503 errors.
		*/
		if err := util.KubeApplyContents(testNamespace, httpbinAllv1, kubeconfig); err != nil {
			t.Errorf("Failed to apply httpbin all v1")
			log.Errorf("Failed to apply httpbin all v1")
		}
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		_, err = util.PodExec(testNamespace, sleepPod, "sleep", "sh -c 'curl  http://httpbin:8080/headers' | python -m json.tool", true, kubeconfig)
		util.Inspect(err, "Failed to get sleep curl response", "", t)

		// check httpbin v1 logs
		v1Pod, err := util.GetPodName(testNamespace, "app=httpbin,version=v1", kubeconfig)
		util.Inspect(err, "Failed to get httpbin v1 pod name", "", t)
		v1msg, err := util.Shell("kubectl logs -n %s --follow=false %s -c %s", testNamespace, v1Pod, "httpbin")
		util.Inspect(err, "Failed to get httpbin v1 log", "", t)
		// check httpbin v2 logs
		v2Pod, err := util.GetPodName(testNamespace, "app=httpbin,version=v2", kubeconfig)
		util.Inspect(err, "Failed to get httpbin v2 pod name", "", t)
		v2msg, err := util.Shell("kubectl logs -n %s --follow=false %s -c %s", testNamespace, v2Pod, "httpbin")
		util.Inspect(err, "Failed to get httpbin v2 log", "", t)
		if strings.Contains(v1msg, "\"GET /headers HTTP/1.1\" 200") && !strings.Contains(v2msg, "\"GET /headers HTTP/1.1\" 200") {
			log.Info("Success. v1 an v2 logs are expected")
		} else {
			t.Errorf("Error. v1 log: %s\n v2 log: %s", v1msg, v2msg)
			log.Errorf("Error. v1 log: %s\n v2 log: %s", v1msg, v2msg)
		}
	})

	t.Run("TrafficManagement_mirroring_traffic_to_v2", func(t *testing.T) {
		defer recoverPanic(t)

		if err := util.KubeApplyContents(testNamespace, httpbinMirrorv2, kubeconfig); err != nil {
			t.Errorf("Failed to apply httpbin mirror v2")
			log.Errorf("Failed to apply httpbin mirror v2")
		}
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		_, err = util.PodExec(testNamespace, sleepPod, "sleep", "sh -c 'curl  http://httpbin:8080/headers' | python -m json.tool", true, kubeconfig)
		util.Inspect(err, "Failed to get sleep curl response", "", t)

		// check httpbin v1 logs
		v1Pod, err := util.GetPodName(testNamespace, "app=httpbin,version=v1", kubeconfig)
		util.Inspect(err, "Failed to get httpbin v1 pod name", "", t)
		v1msg, err := util.Shell("kubectl logs -n %s --follow=false %s -c %s", testNamespace, v1Pod, "httpbin")
		util.Inspect(err, "Failed to get httpbin v1 log", "", t)
		// check httpbin v2 logs
		v2Pod, err := util.GetPodName(testNamespace, "app=httpbin,version=v2", kubeconfig)
		util.Inspect(err, "Failed to get httpbin v2 pod name", "", t)
		v2msg, err := util.Shell("kubectl logs -n %s --follow=false %s -c %s", testNamespace, v2Pod, "httpbin")
		util.Inspect(err, "Failed to get httpbin v2 log", "", t)
		if strings.Contains(v1msg, "\"GET /headers HTTP/1.1\" 200") && strings.Contains(v2msg, "\"GET /headers HTTP/1.1\" 200") {
			log.Info("Success. v1 an v2 logs are expected")
		} else {
			t.Errorf("Error. v1 log: %s\n v2 log: %s", v1msg, v2msg)
		}
	})
}
