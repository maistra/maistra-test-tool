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
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

func cleanupMirroring() {
	log.Log.Info("Cleanup")
	httpbin := examples.Httpbin{"bookinfo"}
	sleep := examples.Sleep{"bookinfo"}
	util.KubeDeleteContents("bookinfo", httpbinAllv1)
	httpbin.UninstallV1()
	httpbin.UninstallV2()
	sleep.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestMirroring(t *testing.T) {
	defer cleanupMirroring()
	defer util.RecoverPanic(t)

	log.Log.Info("TestMirroring")
	httpbin := examples.Httpbin{"bookinfo"}
	sleep := examples.Sleep{"bookinfo"}
	httpbin.InstallV1()
	httpbin.InstallV2()
	sleep.Install()

	t.Run("TrafficManagement_creating_a_default_routing_policy", func(t *testing.T) {
		defer util.RecoverPanic(t)

		if err := util.KubeApplyContents("bookinfo", httpbinAllv1); err != nil {
			t.Errorf("Failed to apply httpbin all v1")
			log.Log.Errorf("Failed to apply httpbin all v1")
		}
		time.Sleep(time.Duration(10) * time.Second)

		sleepPod, err := util.GetPodName("bookinfo", "app=sleep")
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		_, err = util.PodExec("bookinfo", sleepPod, "sleep", `curl -sS http://httpbin:8000/headers`, false)
		util.Inspect(err, "Failed to get sleep curl response", "", t)

		// check httpbin v1 logs
		v1Pod, err := util.GetPodName("bookinfo", "app=httpbin,version=v1")
		util.Inspect(err, "Failed to get httpbin v1 pod name", "", t)
		v1msg, err := util.Shell("kubectl logs -n %s --follow=false %s -c %s", "bookinfo", v1Pod, "httpbin")
		util.Inspect(err, "Failed to get httpbin v1 log", "", t)
		// check httpbin v2 logs
		v2Pod, err := util.GetPodName("bookinfo", "app=httpbin,version=v2")
		util.Inspect(err, "Failed to get httpbin v2 pod name", "", t)
		v2msg, err := util.Shell("kubectl logs -n %s --follow=false %s -c %s", "bookinfo", v2Pod, "httpbin")
		util.Inspect(err, "Failed to get httpbin v2 log", "", t)
		if strings.Contains(v1msg, `"GET /headers HTTP/1.1" 200`) && !strings.Contains(v2msg, `"GET /headers HTTP/1.1" 200`) {
			log.Log.Info("Success. v1 an v2 logs are expected")
		} else {
			t.Errorf("Error. v1 log: %s\n v2 log: %s", v1msg, v2msg)
			log.Log.Errorf("Error. v1 log: %s\n v2 log: %s", v1msg, v2msg)
		}
	})

	t.Run("TrafficManagement_mirroring_traffic_to_v2", func(t *testing.T) {
		defer util.RecoverPanic(t)

		if err := util.KubeApplyContents("bookinfo", httpbinMirrorv2); err != nil {
			t.Errorf("Failed to apply httpbin mirror v2")
			log.Log.Errorf("Failed to apply httpbin mirror v2")
		}
		time.Sleep(time.Duration(10) * time.Second)

		sleepPod, err := util.GetPodName("bookinfo", "app=sleep")
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		_, err = util.PodExec("bookinfo", sleepPod, "sleep", `curl -sS http://httpbin:8000/headers`, false)
		util.Inspect(err, "Failed to get sleep curl response", "", t)

		// check httpbin v1 logs
		v1Pod, err := util.GetPodName("bookinfo", "app=httpbin,version=v1")
		util.Inspect(err, "Failed to get httpbin v1 pod name", "", t)
		v1msg, err := util.Shell("kubectl logs -n %s --follow=false %s -c %s", "bookinfo", v1Pod, "httpbin")
		util.Inspect(err, "Failed to get httpbin v1 log", "", t)
		// check httpbin v2 logs
		v2Pod, err := util.GetPodName("bookinfo", "app=httpbin,version=v2")
		util.Inspect(err, "Failed to get httpbin v2 pod name", "", t)
		v2msg, err := util.Shell("kubectl logs -n %s --follow=false %s -c %s", "bookinfo", v2Pod, "httpbin")
		util.Inspect(err, "Failed to get httpbin v2 log", "", t)
		if strings.Contains(v1msg, `"GET /headers HTTP/1.1" 200`) && strings.Contains(v2msg, `"GET /headers HTTP/1.1" 200`) {
			log.Log.Info("Success. v1 an v2 logs are expected")
		} else {
			t.Errorf("Error. v1 log: %s\n v2 log: %s", v1msg, v2msg)
		}
	})
}
