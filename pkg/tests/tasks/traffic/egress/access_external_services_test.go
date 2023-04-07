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

package egress

import (
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func cleanupAccessExternalServices() {
	log.Log.Info("Cleanup")
	sleep := examples.Sleep{Namespace: "bookinfo"}
	util.KubeDeleteContents("bookinfo", httpbinextTimeout)
	util.KubeDeleteContents("bookinfo", redhatextServiceEntry)
	util.KubeDeleteContents("bookinfo", httpbinextServiceEntry)
	sleep.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestAccessExternalServices(t *testing.T) {
	test.NewTest(t).Id("T11").Groups(test.Full, test.InterOp).NotRefactoredYet()

	defer cleanupAccessExternalServices()
	defer util.RecoverPanic(t)

	log.Log.Info("TestAccessExternalServices")
	sleep := examples.Sleep{Namespace: "bookinfo"}
	sleep.Install()
	sleepPod, err := util.GetPodName("bookinfo", "app=sleep")
	util.Inspect(err, "Failed to get sleep pod name", "", t)

	t.Run("TrafficManagement_egress_envoy_passthrough_to_external_services", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Skip checking the meshConfig outboundTrafficPolicy mode")
		log.Log.Info("make requests to external https services")
		command := `curl -sSI https://www.redhat.com/en | grep  "HTTP/"`
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "200") {
			log.Log.Infof("Success. Get https://www.redhat.com/en response: %s", msg)
		} else {
			log.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}
	})

	t.Run("TrafficManagement_egress_controlled_access_to_external_httpbin_services", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Skip update global.outboundTrafficPolicy.mode")
		log.Log.Info("Create a ServiceEntry to external httpbin")
		util.KubeApplyContents("bookinfo", httpbinextServiceEntry)
		time.Sleep(time.Duration(10) * time.Second)
		command := `curl -sS http://httpbin.org/headers`
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		if err != nil {
			log.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		} else {
			log.Log.Infof("Success. Get http://httpbin.org/headers response:\n%s", msg)
		}
	})

	t.Run("TrafficManagement_egress_access_to_external_https_redhat", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Create a ServiceEntry to external https://www.redhat.com/en")
		util.KubeApplyContents("bookinfo", redhatextServiceEntry)
		time.Sleep(time.Duration(10) * time.Second)
		command := `curl -sSI https://www.redhat.com/en | grep  "HTTP/"`
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "200") {
			log.Log.Infof("Success. Get https://www.redhat.com/en response: %s", msg)
		} else {
			log.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}
	})

	t.Run("TrafficManagement_egress_manage_traffic_to_external_services", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Create a httpbin-ext timeout")
		util.KubeApplyContents("bookinfo", httpbinextTimeout)
		time.Sleep(time.Duration(10) * time.Second)
		command := `time curl -o /dev/null -sS -w "%{http_code}\n" http://httpbin.org/delay/5`
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "504") {
			log.Log.Infof("Get expected response failure: %s", msg)
		} else {
			log.Log.Infof("Error response code: %s", msg)
			t.Errorf("Error response code: %s", msg)
		}
	})
}
