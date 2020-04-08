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
	"fmt"
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/istio/pkg/log"
)

func cleanupAuthPolicy() {
	log.Info("# Cleanup ...")

	util.KubeDeleteContents(meshNamespace, clientRule, kubeconfig)
	

	namespaces := []string{"foo", "bar", "legacy"}
	for _, ns := range namespaces {
		cleanSleep(ns)
		cleanHttpbin(ns)
	}
	time.Sleep(time.Duration(waitTime*2) * time.Second)

}


func TestAuthPolicy(t *testing.T) {
	defer cleanupAuthPolicy()
	defer recoverPanic(t)

	log.Infof("# Authentication Policy")

	// setup
	namespaces := []string{"foo", "bar", "legacy"}
	for _, ns := range namespaces {
		util.CreateOCPNamespace(ns, kubeconfig)
	}

	deployHttpbin("foo")
	deployHttpbin("bar")
	deploySleep("foo")
	deploySleep("bar")
	util.KubeApply("legacy", httpbinLegacyYaml, kubeconfig)
	util.CheckPodRunning("legacy", "app=httpbin", kubeconfig)
	util.KubeApply("legacy", sleepLegacyYaml, kubeconfig)
	util.CheckPodRunning("legacy", "app=sleep", kubeconfig)
	time.Sleep(time.Duration(waitTime*2) * time.Second)

	log.Info("Verify setup")
	for _, from := range namespaces {
		for _, to := range namespaces {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
			util.Inspect(err, "Failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
				to, from, to)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)
			util.Inspect(err, "Failed to get response", "", t)
			if !strings.Contains(msg, "200") {
				t.Errorf("Verify setup -- Unexpected response code: %s", msg)
				log.Errorf("Verify setup -- Unexpected response code: %s", msg)
			} else {
				log.Infof("Success. Get expected response: %s", msg)
			}
		}
	}

	t.Run("Security_authentication_global_mTLS_test", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Globally enabling Istio mutual TLS")
		

		log.Info("Waiting for rules to propagate. Sleep 50 seconds...")
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		ns := []string{"foo", "bar"}
		for _, from := range ns {
			for _, to := range ns {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
				util.Inspect(err, "Failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
					to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)
				util.Inspect(err, "Failed to get response", "", t)
				if !strings.Contains(msg, "503") {
					t.Errorf("Global mTLS expected: 503; Got response code: %s", msg)
					log.Errorf("Global mTLS expected: 503; Got response code: %s", msg)
				} else {
					log.Infof("Response 503 as expected: %s", msg)
				}
			}
		}
		time.Sleep(time.Duration(waitTime) * time.Second)

		util.Inspect(util.KubeApplyContents(meshNamespace, clientRule, kubeconfig), "Failed to apply clientRule", "", t)
		log.Info("Waiting for rules to propagate. Sleep 30 seconds...")
		time.Sleep(time.Duration(waitTime*6) * time.Second)
		for _, from := range ns {
			for _, to := range ns {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
				util.Inspect(err, "Failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
					to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)
				util.Inspect(err, "Failed to get response", "", t)
				if !strings.Contains(msg, "200") {
					t.Errorf("Global mTLS expected: 200; Got response code: %s", msg)
					log.Errorf("Global mTLS expected: 200; Got response code: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
		time.Sleep(time.Duration(waitTime) * time.Second)
	})
	
}