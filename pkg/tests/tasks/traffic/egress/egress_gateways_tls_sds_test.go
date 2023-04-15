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

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func cleanupTLSOriginationSDS(t *testing.T) {
	log.Log.Info("Cleanup")
	util.KubeDeleteContents(meshNamespace, OriginateSDS)
	util.KubeDeleteContents(meshNamespace, meshExternalServiceEntry)
	util.KubeDeleteContents("bookinfo", util.RunTemplate(EgressGatewaySDSTemplate, smcp))
	util.Shell(`kubectl delete -n %s secret client-credential`, meshNamespace)
	util.KubeDeleteContents("bookinfo", util.RunTemplate(ExGatewayTLSFileTemplate, smcp))
	util.KubeDeleteContents("bookinfo", ExServiceEntry)
	app.Uninstall(test.NewTestContext(t), app.Sleep("bookinfo"), app.NginxWithMTLS("mesh-external"))
	time.Sleep(time.Duration(20) * time.Second)
}

func TestTLSOriginationSDS(t *testing.T) {
	test.NewTest(t).Id("T15").Groups(test.Full, test.InterOp).NotRefactoredYet()

	defer cleanupTLSOriginationSDS(t)
	defer util.RecoverPanic(t)

	log.Log.Info("TestEgressGatewaysTLSOrigination SDS")
	app.InstallAndWaitReady(test.NewTestContext(t), app.Sleep("bookinfo")) // replace test.NewTestContext(t) with t when you refactor this test
	sleepPod, _ := util.GetPodName("bookinfo", "app=sleep")

	t.Run("TrafficManagement_egress_gateway_perform_TLS_origination", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Perform TLS origination with an egress gateway")
		util.KubeApplyContents("bookinfo", ExServiceEntry)
		time.Sleep(time.Duration(10) * time.Second)

		command := `curl -sSL -o /dev/null -D - http://istio.io`
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "301 Moved Permanently") {
			log.Log.Info("Success. Get http://istio.io response")
		} else {
			log.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}

		log.Log.Info("Create a Gateway to external istio.io")
		util.KubeApplyContents("bookinfo", util.RunTemplate(ExGatewayTLSFileTemplate, smcp))
		time.Sleep(time.Duration(20) * time.Second)

		command = `curl -sSL -o /dev/null -D - http://istio.io`
		msg, err = util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "301 Moved Permanently") || !strings.Contains(msg, "200") {
			log.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		} else {
			log.Log.Infof("Success. Get http://istio.io response")
		}

		log.Log.Info("Cleanup the TLS origination example")
		util.KubeDeleteContents("bookinfo", util.RunTemplate(ExGatewayTLSFileTemplate, smcp))
		util.KubeDeleteContents("bookinfo", ExServiceEntry)
		time.Sleep(time.Duration(20) * time.Second)
	})

	t.Run("TrafficManagement_egress_gateway_perform_mtls_origination", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Deploy nginx mtls server")

		app.InstallAndWaitReady(test.NewTestContext(t), app.NginxWithMTLS("mesh-external"))

		log.Log.Info("Create client cert secret")
		util.Shell(`kubectl create secret -n %s generic client-credential --from-file=tls.key=%s --from-file=tls.crt=%s --from-file=ca.crt=%s`,
			meshNamespace,
			nginxClientCertKey,
			nginxClientCert,
			nginxServerCACert)

		log.Log.Info("Configure MTLS origination for egress traffic")
		util.KubeApplyContents("bookinfo", util.RunTemplate(EgressGatewaySDSTemplate, smcp))
		util.KubeApplyContents(meshNamespace, meshExternalServiceEntry)
		util.KubeApplyContents(meshNamespace, OriginateSDS)
		time.Sleep(time.Duration(10) * time.Second)

		log.Log.Info("Verify NGINX server")
		cmd := fmt.Sprintf(`curl -sS http://my-nginx.mesh-external.svc.cluster.local`)
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "Welcome to nginx") {
			t.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
			log.Log.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}
	})
}
