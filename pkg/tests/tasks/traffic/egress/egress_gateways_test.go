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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func cleanupEgressGateways() {
	log.Log.Info("Cleanup")
	sleep := examples.Sleep{"bookinfo"}
	util.KubeDeleteContents("bookinfo", util.RunTemplate(ExGatewayHTTPSTemplate, smcp))
	util.KubeDeleteContents("bookinfo", util.RunTemplate(ExGatewayTemplate, smcp))
	util.KubeDeleteContents("bookinfo", ExServiceEntryTLS)
	util.KubeDeleteContents("bookinfo", ExServiceEntry)
	sleep.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestEgressGateways(t *testing.T) {
	test.NewTest(t).Id("T13").Groups(test.Full, test.InterOp).NotRefactoredYet()

	defer cleanupEgressGateways()
	defer util.RecoverPanic(t)

	log.Log.Info("TestEgressGateways")
	sleep := examples.Sleep{"bookinfo"}
	sleep.Install()
	sleepPod, err := util.GetPodName("bookinfo", "app=sleep")
	util.Inspect(err, "Failed to get sleep pod name", "", t)
	proxy, _ := util.GetProxy()
	curlParams := ""
	if proxy.HTTPProxy == "" {
		log.Log.Info("HTTP_PROXY is not set")
	} else {
		curlParams = curlParams + " -x " + proxy.HTTPProxy
	}
	t.Run("TrafficManagement_egress_gateway_for_http_traffic", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Create a ServiceEntry to external istio.io")
		util.KubeApplyContents("bookinfo", ExServiceEntry)
		time.Sleep(time.Duration(10) * time.Second)
		command := fmt.Sprintf(`curl -sSL -o /dev/null %s -D - http://istio.io`, curlParams)
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "301 Moved Permanently") {
			log.Log.Info("Success. Get http://istio.io response")
		} else {
			log.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}

		log.Log.Info("Create a Gateway to external istio.io")
		util.KubeApplyContents("bookinfo", util.RunTemplate(ExGatewayTemplate, smcp))
		time.Sleep(time.Duration(20) * time.Second)

		command = fmt.Sprintf(`curl -sSL -o /dev/null %s -D - http://istio.io`, curlParams)
		msg, err = util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "301 Moved Permanently") {
			log.Log.Infof("Success. Get http://istio.io response: %s", msg)
		} else {
			log.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}

		util.KubeDeleteContents("bookinfo", util.RunTemplate(ExGatewayTemplate, smcp))
		util.KubeDeleteContents("bookinfo", ExServiceEntry)
		time.Sleep(time.Duration(20) * time.Second)
	})

	t.Run("TrafficManagement_egress_gateway_for_https_traffic", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Create a TLS ServiceEntry to external istio.io")
		util.KubeApplyContents("bookinfo", ExServiceEntryTLS)
		time.Sleep(time.Duration(10) * time.Second)

		command := `curl -sSL -o /dev/null -D - https://istio.io`
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "301 Moved Permanently") || !strings.Contains(msg, "200") {
			log.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		} else {
			log.Log.Infof("Success. Get https://istio.io response: %s", msg)
		}

		log.Log.Info("Create a https Gateway to external istio.io")
		util.KubeApplyContents("bookinfo", util.RunTemplate(ExGatewayHTTPSTemplate, smcp))
		time.Sleep(time.Duration(20) * time.Second)

		command = `curl -sSL -o /dev/null -D - https://istio.io`
		msg, err = util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "301 Moved Permanently") || !strings.Contains(msg, "200") {
			log.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		} else {
			log.Log.Infof("Success. Get https://istio.io response: %s", msg)
		}
	})
}
