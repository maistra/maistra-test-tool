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

package authorization

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func cleanupAuthorTCP() {
	log.Log.Info("Cleanup")
	util.KubeDeleteContents("foo", TCPDenyGETPolicy)
	util.KubeDeleteContents("foo", TCPAllowGETPolicy)
	util.KubeDeleteContents("foo", TCPAllowPolicy)
	echo := examples.Echo{Namespace: "foo"}
	echo.UninstallWithProxy()
	sleep := examples.Sleep{Namespace: "foo"}
	sleep.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestAuthorTCP(t *testing.T) {
	test.NewTest(t).Id("T21").Groups(test.Full, test.InterOp).NotRefactoredYet()

	defer cleanupAuthorTCP()
	defer util.RecoverPanic(t)

	log.Log.Info("Authorization for TCP traffic")
	sleep := examples.Sleep{Namespace: "foo"}
	sleep.Install()
	echo := examples.Echo{Namespace: "foo"}
	echo.InstallWithProxy()
	time.Sleep(time.Duration(20) * time.Second)

	log.Log.Info("Verify echo hello port")
	sleepPod, err := util.GetPodName("foo", "app=sleep")
	util.Inspect(err, "Failed to get sleep pod name", "", t)
	ports := []string{"9000", "9001", "9002"}
	for _, port := range ports {
		if port == "9000" || port == "9001" {
			cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc tcp-echo %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`, port, port)
			msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true)
			util.Inspect(err, "Failed to get response", "", t)
			if !strings.Contains(msg, "connection succeeded") {
				log.Log.Errorf("Verify setup Unexpected response: %s", msg)
				t.Errorf("Verify setup Unexpected response: %s", msg)
			} else {
				log.Log.Infof("Success. Get expected response: %s", msg)
			}
		} else {
			tcpEchoPod, _ := util.GetPodName("foo", "app=tcp-echo")
			podIP, _ := util.Shell(`kubectl get pod %s -n foo -o jsonpath="{.status.podIP}"`, tcpEchoPod)
			cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc %s %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`, port, podIP, port)
			msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true)
			util.Inspect(err, "Failed to get response", "", t)
			if !strings.Contains(msg, "connection succeeded") {
				log.Log.Errorf("Verify setup Unexpected response: %s", msg)
				t.Errorf("Verify setup Unexpected response: %s", msg)
			} else {
				log.Log.Infof("Success. Get expected response: %s", msg)
			}
		}
	}

	t.Run("Security_authorization_rbac_allow_GET_tcp", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Apply a policy to allow requests to port 9000 and 9001")
		util.KubeApplyContents("foo", TCPAllowPolicy)
		time.Sleep(time.Duration(10) * time.Second)

		ports := []string{"9000", "9001", "9002"}
		for _, port := range ports {
			if port == "9000" || port == "9001" {
				cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc tcp-echo %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`, port, port)
				msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true)
				util.Inspect(err, "Failed to get response", "", t)
				if !strings.Contains(msg, "connection succeeded") {
					log.Log.Errorf("Verify allow GET Unexpected response: %s", msg)
					t.Errorf("Verify allow GET Unexpected response: %s", msg)
				} else {
					log.Log.Infof("Success. Get expected response: %s", msg)
				}
			} else {
				tcpEchoPod, _ := util.GetPodName("foo", "app=tcp-echo")
				podIP, _ := util.Shell(`kubectl get pod %s -n foo -o jsonpath="{.status.podIP}"`, tcpEchoPod)
				cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc %s %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`, port, podIP, port)
				msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true)
				util.Inspect(err, "Failed to get response", "", t)
				if !strings.Contains(msg, "connection rejected") {
					log.Log.Errorf("Verify allow GET Unexpected response: %s", msg)
					t.Errorf("Verify allow GET Unexpected response: %s", msg)
				} else {
					log.Log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
	})

	t.Run("Security_authorization_rbac_invalid_policy_tcp", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Apply a policy to allow requests to port 9000 and add an HTTP GET field")
		util.KubeApplyContents("foo", TCPAllowGETPolicy)
		time.Sleep(time.Duration(10) * time.Second)

		ports := []string{"9000", "9001"}
		for _, port := range ports {
			cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc tcp-echo %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`, port, port)
			msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true)
			util.Inspect(err, "Failed to get response", "", t)
			if !strings.Contains(msg, "connection rejected") {
				log.Log.Errorf("Verify invalid rule Unexpected response: %s", msg)
				t.Errorf("Verify invalid rule Unexpected response: %s", msg)
			} else {
				log.Log.Infof("Success. Get expected response: %s", msg)
			}
		}
	})

	t.Run("Security_authorization_rbac_deny_GET_tcp", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Apply a DENY policy")
		util.KubeApplyContents("foo", TCPDenyGETPolicy)
		time.Sleep(time.Duration(10) * time.Second)

		ports := []string{"9000", "9001"}
		for _, port := range ports {
			cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc tcp-echo %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`, port, port)
			msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true)
			util.Inspect(err, "Failed to get response", "", t)

			if port == "9000" {
				if !strings.Contains(msg, "connection rejected") {
					log.Log.Errorf("Verify DENY rule Unexpected response: %s", msg)
					t.Errorf("Verify DENY rule Unexpected response: %s", msg)
				} else {
					log.Log.Infof("Success. Get expected response: %s", msg)
				}
			}
			if port == "9001" {
				if !strings.Contains(msg, "connection succeeded") {
					log.Log.Errorf("Verify DENY rule Unexpected response: %s", msg)
					t.Errorf("Verify DENY rule Unexpected response: %s", msg)
				} else {
					log.Log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
	})
}
