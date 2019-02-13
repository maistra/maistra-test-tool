// Copyright 2019 Istio Authors
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

// Package dashboard provides testing of the grafana dashboards used in Istio
// to provide mesh monitoring capabilities.

package maistra

import (
	"fmt"
	"regexp"
	"strings"
	"strconv"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)

func cleanup12(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDelete(namespace, httpbinFortioYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinCircuitBreakerYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinYaml, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}


func configHttpbinCircuitBreaker(namespace, kubeconfig string) error {
	if err := util.KubeApply(namespace, httpbinCircuitBreakerYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}



func Test12(t *testing.T) {
	log.Info("# TC_12 Circuit Breaking")
	Inspect(deployHttpbin(testNamespace, kubeconfigFile), "failed to deploy httpbin", "", t)
	Inspect(configHttpbinCircuitBreaker(testNamespace, kubeconfigFile), "failed to apply rule", "", t)
	Inspect(deployFortio(testNamespace, kubeconfigFile), "failed to deploy fortio", "", t)

	// trip breaker
	pod, err := util.GetPodName(testNamespace, "app=fortio", kubeconfigFile)
	Inspect(err, "failed to get fortio pod", "", t)
	command := "load -curl  http://httpbin:8000/get"
	msg, err := util.PodExec(testNamespace, pod, "fortio /usr/local/bin/fortio", command, false, kubeconfigFile)
	Inspect(err, "failed to get response", "", t)
	if strings.Contains(msg, "200 OK") {
		log.Infof("Success. Get correct response")
	} else {
		t.Errorf("Error response")
	}
	
	log.Info("# Tripping the circuit breaker")
	connection := 4
	reqCount := 20
	tolerance := 0.30
	
	command = fmt.Sprintf("load -c %d -qps 0 -n %d -loglevel Warning http://httpbin:8000/get", connection, reqCount)
	msg, err = util.PodExec(testNamespace, pod, "fortio /usr/local/bin/fortio", command, false, kubeconfigFile)
	Inspect(err, "failed to get response", "", t)

	re := regexp.MustCompile(`Code 200.*`)
	line := re.FindStringSubmatch(msg)[0]
	re = regexp.MustCompile(`: [\d]+`)
	word := re.FindStringSubmatch(line)[0]
	c200, err := strconv.Atoi(strings.TrimLeft(word, ": "))
	Inspect(err, "failed to parse code 200 count", "", t)

	re = regexp.MustCompile(`Code 503.*`)
	line = re.FindStringSubmatch(msg)[0]
	re = regexp.MustCompile(`: [\d]+`)
	word = re.FindStringSubmatch(line)[0]
	c503, err := strconv.Atoi(strings.TrimLeft(word, ": "))
	Inspect(err, "failed to parse code 503 count", "", t)
	
	if isWithinPercentage(c200, reqCount, 0.5, tolerance) && isWithinPercentage(c503, reqCount, 0.5, tolerance) {
		log.Infof(
			"Success. Circuit breaking acts as expected. " +
			"Code 200 hit %d, Code 503 hit %d", c200, c503)
	} else {
		t.Errorf(
			"Failed Circuit breaking. " +
			"Code 200 hit %d, Code 503 hit %d", c200, c503)
	}

	defer cleanup12(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			log.Infof("Test failed: %v", err)
		}
	}()
}