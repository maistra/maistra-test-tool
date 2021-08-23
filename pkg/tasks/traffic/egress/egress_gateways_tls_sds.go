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
)

func cleanupTLSOriginationSDS() {
	util.Log.Info("Cleanup")
	util.KubeDeleteContents("istio-system", OriginateSDS)
	util.KubeDeleteContents("bookinfo", EgressGatewaySDS)
	util.Shell(`kubectl delete -n %s secret client-credential`, "istio-system")
	util.Shell(`kubectl delete -n %s secret client-credential-cacert`, "istio-system")
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

	nginx := examples.Nginx{"mesh-external"}
	nginx.Install("../testdata/examples/x86/nginx/nginx_mesh_external.conf")
	util.Shell(`kubectl create -n %s secret generic client-credential-cacert --from-file=%s`, "istio-system", nginxServerCACert)

	t.Run("TrafficManagement_egress_configure_tls_origination", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Configure simple TLS origination for egress traffic")
		util.KubeApplyContents("bookinfo", EgressGatewaySDS)
		time.Sleep(time.Duration(20) * time.Second)
		util.KubeApplyContents("istio-system", OriginateSDS)
		time.Sleep(time.Duration(10) * time.Second)

		util.Log.Info("Verify NGINX server")
		cmd := fmt.Sprintf(`curl -sS http://my-nginx.mesh-external.svc.cluster.local`)
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "Welcome to nginx") {
			t.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
			util.Log.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
		} else {
			util.Log.Infof("Success. Get expected response: %s", msg)
		}
	})

	cleanupTLSOriginationSDS()

	t.Run("TrafficManagement_egress_configure_mtls_origination", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Configure mutual TLS origination for egress traffic")
		nginx := examples.Nginx{"mesh-external"}
		nginx.Install("../testdata/examples/x86/nginx/nginx_mesh_external_ssl.conf")
		util.Shell(`kubectl create -n %s secret generic client-credential --from-file=tls.key=%s --from-file=tls.crt=%s --from-file=ca.crt=%s`,
			"istio-system", nginxClientCertKey, nginxClientCert, nginxServerCACert)

	})
}
