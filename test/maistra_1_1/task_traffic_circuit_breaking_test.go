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
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"maistra/util"
)


func cleanupCircuitBreaking(namespace string) {
	log.Info("# Cleanup ...")
	cleanFortio(namespace)
	util.KubeDeleteContents(namespace, httpbinCircuitBreaker, kubeconfig)
	cleanHttpbin(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestCircuitBreaking(t *testing.T) {
	defer cleanupCircuitBreaking(testNamespace)
	defer recoverPanic(t)

	log.Info("# TC_12 Circuit Breaking")
	deployHttpbin(testNamespace)

	/*
	If you installed/configured Istio with mutual TLS authentication enabled, you must add a TLS traffic policy mode: ISTIO_MUTUAL to the DestinationRule before applying it. Otherwise requests will generate 503 errors as described here.
	*/
	if err := util.KubeApplyContents(testNamespace, httpbinCircuitBreaker, kubeconfig); err != nil {
		t.Errorf("Failed to configure circuit breaker")
		log.Errorf("Failed to configure circuit breaker")
	}
	time.Sleep(time.Duration(waitTime) * time.Second)

	deployFortio(testNamespace)

	t.Run("Tripping_the_circuit_breaker", func(t *testing.T) {
		defer recoverPanic(t)

		// trip breaker
		pod, err := util.GetPodName(testNamespace, "app=fortio", kubeconfig)
		util.Inspect(err, "failed to get fortio pod", "", t)

		command := "load -curl  http://httpbin:8000/get"
		msg, err := util.PodExec(testNamespace, pod, "fortio /usr/bin/fortio", command, false, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "200 OK") {
			log.Infof("Success. Get correct response")
		} else {
			t.Errorf("Error response: %v", msg)
			log.Errorf("Error response: %v", msg)
		}

		log.Info("# Tripping the circuit breaker")
		connection := 3
		reqCount := 30
		tolerance := 0.20

		command = fmt.Sprintf("load -c %d -qps 0 -n %d -loglevel Warning http://httpbin:8000/get", connection, reqCount)
		msg, err = util.PodExec(testNamespace, pod, "fortio /usr/bin/fortio", command, false, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)

		re := regexp.MustCompile(`Code 200.*`)
		line := re.FindStringSubmatch(msg)[0]
		re = regexp.MustCompile(`: [\d]+`)
		word := re.FindStringSubmatch(line)[0]
		c200, err := strconv.Atoi(strings.TrimLeft(word, ": "))
		util.Inspect(err, "Failed to parse code 200 count", "", t)

		re = regexp.MustCompile(`Code 503.*`)
		line = re.FindStringSubmatch(msg)[0]
		re = regexp.MustCompile(`: [\d]+`)
		word = re.FindStringSubmatch(line)[0]
		c503, err := strconv.Atoi(strings.TrimLeft(word, ": "))
		util.Inspect(err, "Failed to parse code 503 count", "", t)

		if isWithinPercentage(c200, reqCount, 0.6, tolerance) && isWithinPercentage(c503, reqCount, 0.4, tolerance) {
			log.Infof(
				"Success. Circuit breaking acts as expected. "+
					"Code 200 hit %d of %d, Code 503 hit %d of %d", c200, reqCount, c503, reqCount)
		} else {
			t.Errorf(
				"Failed Circuit breaking. "+
					"Code 200 hit %d 0f %d, Code 503 hit %d of %d", c200, reqCount, c503, reqCount)
		}
	})
}
