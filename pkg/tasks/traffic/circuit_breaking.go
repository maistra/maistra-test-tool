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

package traffic

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

func cleanupCircuitBreaking() {
	log.Log.Info("Cleanup")
	util.KubeDeleteContents("bookinfo", httpbinCircuitBreaker)
	fortio := examples.Fortio{"bookinfo"}
	httpbin := examples.Httpbin{"bookinfo"}
	fortio.Uninstall()
	httpbin.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestCircuitBreaking(t *testing.T) {
	defer cleanupCircuitBreaking()
	defer util.RecoverPanic(t)

	log.Log.Info("TestCircuitBreaking")
	fortio := examples.Fortio{"bookinfo"}
	httpbin := examples.Httpbin{"bookinfo"}
	httpbin.Install()
	fortio.Install()

	t.Run("TrafficManagement_tripping_circuit_breaker", func(t *testing.T) {
		defer util.RecoverPanic(t)

		if err := util.KubeApplyContents("bookinfo", httpbinCircuitBreaker); err != nil {
			t.Errorf("Failed to configure circuit breaker")
			log.Log.Errorf("Failed to configure circuit breaker")
		}
		time.Sleep(time.Duration(10) * time.Second)

		// verify curl
		pod, err := util.GetPodName("bookinfo", "app=fortio")
		util.Inspect(err, "failed to get fortio pod", "", t)

		command := `/usr/bin/fortio curl -quiet http://httpbin:8000/get`
		msg, err := util.PodExec("bookinfo", pod, "fortio", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "200 OK") {
			log.Log.Infof("Success. Get correct response")
		} else {
			t.Errorf("Error response: %v", msg)
			log.Log.Errorf("Error response: %v", msg)
		}

		log.Log.Info("Tripping the circuit breaker")
		connection := 2
		reqCount := 50
		tolerance := 0.5

		command = fmt.Sprintf(`/usr/bin/fortio load -c %d -qps 0 -n %d -loglevel Warning http://httpbin:8000/get`, connection, reqCount)
		msg, err = util.PodExec("bookinfo", pod, "fortio", command, false)
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

		if util.IsWithinPercentage(c200, reqCount, 0.6, tolerance) && util.IsWithinPercentage(c503, reqCount, 0.4, tolerance) {
			log.Log.Infof(
				"Success. Circuit breaking acts as expected. "+
					"Code 200 hit %d of %d, Code 503 hit %d of %d", c200, reqCount, c503, reqCount)
		} else {
			t.Errorf(
				"Failed Circuit breaking. "+
					"Code 200 hit %d 0f %d, Code 503 hit %d of %d", c200, reqCount, c503, reqCount)
		}

		log.Log.Info("Query the istio-proxy stats")
		command = fmt.Sprintf(`pilot-agent request GET stats | grep httpbin | grep pending`)
		msg, err = util.PodExec("bookinfo", pod, "istio-proxy", command, false)
		log.Log.Infof("%s", msg)
	})
}
