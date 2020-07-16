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

func cleanupTCPShifting(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDelete(namespace, echoAllv1Yaml, kubeconfig)
	cleanEcho(namespace)
	cleanSleep(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestTCPShifting(t *testing.T) {

	defer cleanupTCPShifting(testNamespace)
	defer recoverPanic(t)

	log.Infof("# TestTCPShifting")

	deploySleep(testNamespace)
	deployEcho(testNamespace)

	t.Run("TrafficManagement_100_percent_v1_tcp_shift_test", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("# Shifting all TCP traffic to v1")
		util.KubeApply(testNamespace, echoAllv1Yaml, kubeconfig)

		if err := util.KubeApplyContents(testNamespace, tcpEchoAllv1, kubeconfig); err != nil {
			t.Errorf("Failed to shift traffic to all v1")
			log.Errorf("Failed to shift traffic to all v1")
		}
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf("sh -c \"(date; sleep 1) | nc %s %s\"", "tcp-echo", "9000")
		for i := 0; i < 20; i++ {
			msg, err := util.PodExec(testNamespace, sleepPod, "sleep", cmd, true, kubeconfig)
			util.Inspect(err, "Failed to get response", "", t)
			if !strings.Contains(msg, "one") {
				t.Errorf("echo one; Got response: %s", msg)
				log.Errorf("echo one; Got response: %s", msg)
			} else {
				log.Infof("%s", msg)
			}
		}
	})

	t.Run("TrafficManagement_20_percent_v2_tcp_shift_test", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("# Shifting 20% TCP traffic to v2 tolerance 15% ")
		if err := util.KubeApplyContents(testNamespace, tcpEcho20v2, kubeconfig); err != nil {
			t.Errorf("Failed to shift traffic to 20 percent v2")
			log.Errorf("Failed to shift traffic to 20 percent v2")
		}
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		tolerance := 0.15
		totalShot := 60
		c1, c2 := 0, 0

		sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf("sh -c \"(date; sleep 1) | nc %s %s\"", "tcp-echo", "9000")

		for i := 0; i < totalShot; i++ {
			msg, err := util.PodExec(testNamespace, sleepPod, "sleep", cmd, true, kubeconfig)
			util.Inspect(err, "Failed to get response", "", t)
			if strings.Contains(msg, "one") {
				c1++
			} else if strings.Contains(msg, "two") {
				c2++
			} else {
				log.Errorf("Unexpected echo version: %s", msg)
			}
		}
		if isWithinPercentage(c1, totalShot, 0.8, tolerance) && isWithinPercentage(c2, totalShot, 0.2, tolerance) {
			log.Infof("Success. Traffic shifting acts as expected. "+
				"v1 version hit %d, v2 version hit %d", c1, c2)
		} else {
			t.Errorf("Failed traffic shifting test for 20 percent. "+
				"v1 version hit %d, v2 version hit %d", c1, c2)
		}
	})
}
