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

func cleanupTCPShifting(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDelete(namespace, echoAllv1Yaml, kubeconfig)
	cleanEcho(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func checkEcho(ingressHost, ingressTCPPort string) (string, error) {
	msg, err := util.ShellSilent("docker run -e INGRESS_HOST=%s -e INGRESS_PORT=%s --rm busybox sh -c \"(date; sleep 1) | nc %s %s\"",
		ingressHost, ingressTCPPort, ingressHost, ingressTCPPort)
	if err != nil {
		return "", err
	}
	return msg, nil
}

func TestTCPShifting(t *testing.T) {

	defer cleanupTCPShifting(testNamespace)
	defer recoverPanic(t)

	log.Infof("# TestTCPShifting")
	tcpPort, _ := util.GetTCPIngressPort(meshNamespace, "istio-ingressgateway", kubeconfig)

	deployEcho(testNamespace)

	t.Run("TrafficManagement_100_percent_v1_tcp_shift_test", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("# Shifting all TCP traffic to v1")
		if err := util.KubeApply(testNamespace, echoAllv1Yaml, kubeconfig); err != nil {
			t.Errorf("Failed to shift traffic to all v1")
			log.Errorf("Failed to shift traffic to all v1")
		}
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		tolerance := 0.1
		totalShot := 10
		versionCount := 0

		log.Infof("Waiting for checking echo dates. Sleep %d seconds...", totalShot*1)

		for i := 0; i < totalShot; i++ {
			time.Sleep(time.Duration(1) * time.Second)
			msg, err := checkEcho(gatewayHTTP, tcpPort)
			if err != nil {
				msg, err = checkEcho(gatewayHTTP, tcpPort)
			}
			util.Inspect(err, "Faild to get date", "", t)
			if strings.Contains(msg, "one") {
				versionCount++
			} else {
				log.Errorf("Unexpected echo version: %s", msg)
			}
		}

		if isWithinPercentage(versionCount, totalShot, 1, tolerance) {
			log.Info("Success. TCP Traffic shifting acts as expected for 100 percent.")
		} else {
			t.Errorf(
				"Failed traffic shifting test for 100 percent. "+
					"Expected version hit %d", versionCount)
		}
	})

	t.Run("TrafficManagement_20_percent_v2_tcp_shift_test", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("# Shifting 20% TCP traffic to v2 tolerance 10% ")
		if err := util.KubeApply(testNamespace, echo20v2Yaml, kubeconfig); err != nil {
			t.Errorf("Failed to shift traffic to 20 percent v2")
			log.Errorf("Failed to shift traffic to 20 percent v2")
		}
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		tolerance := 0.15
		totalShot := 60
		c1, c2 := 0, 0

		log.Infof("Waiting for checking echo dates. Sleep %d seconds...", totalShot*2)

		for i := 0; i < totalShot; i++ {
			time.Sleep(time.Duration(2) * time.Second)
			msg, err := checkEcho(gatewayHTTP, tcpPort)
			if err != nil {
				msg, err = checkEcho(gatewayHTTP, tcpPort)
			}
			util.Inspect(err, "Failed to get date", "", t)
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
