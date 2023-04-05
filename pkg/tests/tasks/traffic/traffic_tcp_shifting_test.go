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
	_ "embed"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	// OSSM need custom changes in VirtualService tcp-echo
	//go:embed yaml/tcp-echo-all-v1.yaml
	echoAllv1Yaml string

	//go:embed yaml/tcp-echo-20-v2.yaml
	echo20v2Yaml string
)

func cleanupTCPShifting() {
	log.Log.Info("Cleanup")
	echo := examples.Echo{Namespace: "bookinfo"}
	sleep := examples.Sleep{Namespace: "bookinfo"}
	util.KubeDeleteContents("bookinfo", echoAllv1Yaml)
	echo.Uninstall()
	sleep.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestTCPShifting(t *testing.T) {
	test.NewTest(t).Id("T4").Groups(test.Full, test.InterOp).NotRefactoredYet()

	defer cleanupTCPShifting()
	defer util.RecoverPanic(t)

	log.Log.Info("TestTCPShifting")
	echo := examples.Echo{Namespace: "bookinfo"}
	sleep := examples.Sleep{Namespace: "bookinfo"}
	echo.Install()
	sleep.Install()

	t.Run("TrafficManagement_100_percent_v1_tcp_shift_test", func(t *testing.T) {
		defer util.RecoverPanic(t)
		log.Log.Info("Shifting all TCP traffic to v1")
		util.KubeApplyContents("bookinfo", echoAllv1Yaml)

		sleepPod, err := util.GetPodName("bookinfo", "app=sleep")
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf(`sh -c "(date; sleep 1) | nc %s %s"`, "tcp-echo", "9000")
		for i := 0; i < 20; i++ {
			msg, err := util.PodExec("bookinfo", sleepPod, "sleep", cmd, true)
			util.Inspect(err, "Failed to get response", "", t)
			if !strings.Contains(msg, "one") {
				t.Errorf("echo one; Got response: %s", msg)
				log.Log.Errorf("echo one; Got response: %s", msg)
			} else {
				log.Log.Infof("%s", msg)
			}
		}
	})

	t.Run("TrafficManagement_20_percent_v2_tcp_shift_test", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Shifting 20%% TCP traffic to v2 tolerance 15%%")
		util.KubeApplyContents("bookinfo", echo20v2Yaml)

		tolerance := 0.15
		totalShot := 60
		c1, c2 := 0, 0

		sleepPod, err := util.GetPodName("bookinfo", "app=sleep")
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf(`sh -c "(date; sleep 1) | nc %s %s"`, "tcp-echo", "9000")

		for i := 0; i < totalShot; i++ {
			msg, err := util.PodExec("bookinfo", sleepPod, "sleep", cmd, true)
			util.Inspect(err, "Failed to get response", "", t)
			if strings.Contains(msg, "one") {
				c1++
			} else if strings.Contains(msg, "two") {
				c2++
			} else {
				log.Log.Errorf("Unexpected echo version: %s", msg)
			}
		}
		if util.IsWithinPercentage(c1, totalShot, 0.8, tolerance) && util.IsWithinPercentage(c2, totalShot, 0.2, tolerance) {
			log.Log.Infof("Success. Traffic shifting acts as expected. "+
				"v1 version hit %d, v2 version hit %d", c1, c2)
		} else {
			t.Errorf("Failed traffic shifting test for 20 percent. "+
				"v1 version hit %d, v2 version hit %d", c1, c2)
		}
	})
}
