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
)

func cleanupTLSOriginationSDS() {
	util.Log.Info("Cleanup")
	util.KubeDeleteContents(meshNamespace, OriginateSDS)
	util.KubeDeleteContents("mesh-external", util.RunTemplate(EgressGatewaySDSTemplate, smcp))
	util.Shell(`kubectl delete -n %s secret client-credential`, meshNamespace)
	util.KubeDeleteContents("bookinfo", util.RunTemplate(ExGatewayTLSFileTemplate, smcp))
	util.KubeDeleteContents("bookinfo", ExServiceEntry)
	sleep := examples.Sleep{"bookinfo"}
	nginx := examples.Nginx{"mesh-external"}
	sleep.Uninstall()
	nginx.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestTLSOriginationSDS(t *testing.T) {
	defer cleanupTLSOriginationSDS()
	defer util.RecoverPanic(t)

	util.Log.Info("TestEgressGatewaysTLSOrigination SDS")
	sleep := examples.Sleep{"bookinfo"}
	sleep.Install()
	sleepPod, _ := util.GetPodName("bookinfo", "app=sleep")

	t.Run("TrafficManagement_egress_gateway_perform_TLS_origination", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Perform TLS origination with an egress gateway")
		util.KubeApplyContents("bookinfo", ExServiceEntry)
		time.Sleep(time.Duration(10) * time.Second)

		command := `curl -sSL -o /dev/null -D - http://istio.io`
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "301 Moved Permanently") {
			util.Log.Info("Success. Get http://istio.io response")
		} else {
			util.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}

		util.Log.Info("Create a Gateway to external istio.io")
		util.KubeApplyContents("bookinfo", util.RunTemplate(ExGatewayTLSFileTemplate, smcp))
		time.Sleep(time.Duration(20) * time.Second)

		command = `curl -sSL -o /dev/null -D - http://istio.io`
		msg, err = util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "301 Moved Permanently") || !strings.Contains(msg, "200") {
			util.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		} else {
			util.Log.Infof("Success. Get http://istio.io response")
		}

		util.Log.Info("Cleanup the TLS origination example")
		util.KubeDeleteContents("bookinfo", util.RunTemplate(ExGatewayTLSFileTemplate, smcp))
		util.KubeDeleteContents("bookinfo", ExServiceEntry)
		time.Sleep(time.Duration(20) * time.Second)
	})
}
