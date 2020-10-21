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
	"fmt"
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupAuthorizationTCP() {
	log.Info("# Cleanup ...")

	util.KubeDeleteContents("foo", tcpPolicyAllow, kubeconfig)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
	cleanEchoWithProxy("foo")
	cleanSleep("foo")
}

func TestAuthorizationTCP(t *testing.T) {
	defer cleanupAuthorizationTCP()
	defer recoverPanic(t)

	log.Info("Authorization for TCP traffic")

	deploySleep("foo")
	deployEchoWithProxy("foo")

	log.Info("Verify echo hello port")
	sleepPod, err := util.GetPodName("foo", "app=sleep", kubeconfig)
	util.Inspect(err, "Failed to get sleep pod name", "", t)

	ports := []string{"9000", "9001", "9002"}
	for _, port := range ports {
		if port == "9000" || port == "9001" {
			cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc tcp-echo %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`, port, port)
			msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
			util.Inspect(err, "Failed to get response", "", t)
			if !strings.Contains(msg, "connection succeeded") {
				log.Errorf("Verify setup Unexpected response: %s", msg)
				t.Errorf("Verify setup Unexpected response: %s", msg)
			} else {
				log.Infof("Success. Get expected response: %s", msg)
			}
		} else {
			tcpEchoPod, err := util.GetPodName("foo", "app=tcp-echo", kubeconfig)
			podIP, err := util.Shell(`kubectl get pod %s -n foo -o jsonpath="{.status.podIP}"`, tcpEchoPod)
			cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc %s %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`, port, podIP, port)
			msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
			util.Inspect(err, "Failed to get response", "", t)
			if !strings.Contains(msg, "connection succeeded") {
				log.Errorf("Verify setup Unexpected response: %s", msg)
				t.Errorf("Verify setup Unexpected response: %s", msg)
			} else {
				log.Infof("Success. Get expected response: %s", msg)
			}
		}
	}

	t.Run("Security_authorization_rbac_allow_GET_tcp", func(t *testing.T) {
		defer recoverPanic(t)

		util.KubeApplyContents("foo", tcpPolicyAllow, kubeconfig)
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		ports := []string{"9000", "9001", "9002"}
		for _, port := range ports {
			if port == "9000" || port == "9001" {
				cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc tcp-echo %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`, port, port)
				msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
				util.Inspect(err, "Failed to get response", "", t)
				if !strings.Contains(msg, "connection succeeded") {
					log.Errorf("Verify allow GET Unexpected response: %s", msg)
					t.Errorf("Verify allow GET Unexpected response: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			} else {
				tcpEchoPod, err := util.GetPodName("foo", "app=tcp-echo", kubeconfig)
				podIP, err := util.Shell(`kubectl get pod %s -n foo -o jsonpath="{.status.podIP}"`, tcpEchoPod)
				cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc %s %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`, port, podIP, port)
				msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
				util.Inspect(err, "Failed to get response", "", t)
				if !strings.Contains(msg, "connection rejected") {
					log.Errorf("Verify allow GET Unexpected response: %s", msg)
					t.Errorf("Verify allow GET Unexpected response: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
	})

	t.Run("Security_authorization_rbac_invalid_policy_tcp", func(t *testing.T) {
		defer recoverPanic(t)

		util.KubeApplyContents("foo", tcpPolicyInvalid, kubeconfig)
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		ports := []string{"9000", "9001"}
		for _, port := range ports {
			cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc tcp-echo %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`, port, port)
			msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
			util.Inspect(err, "Failed to get response", "", t)
			if !strings.Contains(msg, "connection rejected") {
				log.Errorf("Verify invalid rule Unexpected response: %s", msg)
				t.Errorf("Verify invalid rule Unexpected response: %s", msg)
			} else {
				log.Infof("Success. Get expected response: %s", msg)
			}
		}
	})

	t.Run("Security_authorization_rbac_deny_GET_tcp", func(t *testing.T) {
		defer recoverPanic(t)

		util.KubeApplyContents("foo", tcpPolicyDeny, kubeconfig)
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		ports := []string{"9000", "9001"}
		for _, port := range ports {
			cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc tcp-echo %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`, port, port)
			msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
			util.Inspect(err, "Failed to get response", "", t)

			if port == "9000" {
				if !strings.Contains(msg, "connection rejected") {
					log.Errorf("Verify DENY rule Unexpected response: %s", msg)
					t.Errorf("Verify DENY rule Unexpected response: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
			if port == "9001" {
				if !strings.Contains(msg, "connection succeeded") {
					log.Errorf("Verify DENY rule Unexpected response: %s", msg)
					t.Errorf("Verify DENY rule Unexpected response: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
	})
}
