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

var (
	gatewayPatchAdd = `
[{
"op": "add",
"path": "/spec/template/spec/containers/0/volumeMounts/0",
"value": {
"mountPath": "/etc/istio/nginx-client-certs",
"name": "nginx-client-certs",
"readOnly": true
}
},
{
"op": "add",
"path": "/spec/template/spec/volumes/0",
"value": {
"name": "nginx-client-certs",
"secret": {
"secretName": "nginx-client-certs",
"optional": true
}
}
},
{
"op": "add",
"path": "/spec/template/spec/containers/0/volumeMounts/1",
"value": {
"mountPath": "/etc/istio/nginx-ca-certs",
"name": "nginx-ca-certs",
"readOnly": true
}
},
{
"op": "add",
"path": "/spec/template/spec/volumes/1",
"value": {
"name": "nginx-ca-certs",
"secret": {
"secretName": "nginx-ca-certs",
"optional": true
}
}
}]
`
)

func cleanupTLSOriginationFileMount() {
	util.Log.Info("Cleanup")
	sleep := examples.Sleep{"bookinfo"}
	nginx := examples.Nginx{"bookinfo"}
	util.KubeDeleteContents(meshNamespace, nginxMeshRule)
	util.KubeDeleteContents("bookinfo", util.RunTemplate(nginxGatewayTLSTemplate, smcp))

	util.Shell(`kubectl -n %s rollout undo deploy istio-egressgateway`, meshNamespace)
	time.Sleep(time.Duration(20) * time.Second)
	util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 180s`, meshNamespace)
	util.Shell(`kubectl -n %s rollout history deploy istio-egressgateway`, meshNamespace)

	util.Shell(`kubectl delete -n %s secret nginx-client-certs`, meshNamespace)
	util.Shell(`kubectl delete -n %s secret nginx-ca-certs`, meshNamespace)
	util.KubeDeleteContents("bookinfo", util.RunTemplate(ExGatewayTLSFileTemplate, smcp))
	util.KubeDeleteContents("bookinfo", ExServiceEntry)
	nginx.Uninstall()
	sleep.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestTLSOriginationFileMount(t *testing.T) {
	defer cleanupTLSOriginationFileMount()
	defer util.RecoverPanic(t)

	util.Log.Info("TestEgressGatewaysTLSOrigination File Mount")
	sleep := examples.Sleep{"bookinfo"}
	sleep.Install()
	sleepPod, err := util.GetPodName("bookinfo", "app=sleep")
	util.Inspect(err, "Failed to get sleep pod name", "", t)

	nginx := examples.Nginx{"bookinfo"}
	nginx.Install("../testdata/examples/x86/nginx/nginx_ssl.conf")

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

	t.Run("TrafficManagement_egress_gateway_perform_MTLS_origination", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Redeploy the egress gateway with the client certs")
		util.Shell(`kubectl create -n %s secret tls nginx-client-certs --key %s --cert %s`, meshNamespace, nginxClientCertKey, nginxClientCert)
		util.Shell(`kubectl create -n %s secret generic nginx-ca-certs --from-file=%s`, meshNamespace, nginxServerCACert)

		util.Log.Info("Patch egress gateway")
		util.Shell(`kubectl -n %s rollout history deploy istio-egressgateway`, meshNamespace)
		util.Shell(`kubectl -n %s patch --type=json deploy istio-egressgateway -p='%s'`, meshNamespace, strings.ReplaceAll(gatewayPatchAdd, "\n", ""))
		time.Sleep(time.Duration(20) * time.Second)
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 180s`, meshNamespace)
		util.Log.Info("Verify the istio-egressgateway pod")
		util.Shell(`kubectl exec -n %s "$(kubectl -n %s get pods -l %s -o jsonpath='{.items[0].metadata.name}')" -- ls -al %s %s`,
			meshNamespace, meshNamespace,
			"istio=egressgateway",
			"/etc/istio/nginx-client-certs",
			"/etc/istio/nginx-ca-certs")
		util.Shell(`kubectl -n %s rollout history deploy istio-egressgateway`, meshNamespace)

		util.Log.Info("Configure MTLS origination for egress traffic")
		util.KubeApplyContents("bookinfo", util.RunTemplate(nginxGatewayTLSTemplate, smcp))
		time.Sleep(time.Duration(20) * time.Second)
		util.KubeApplyContents(meshNamespace, nginxMeshRule)
		time.Sleep(time.Duration(10) * time.Second)

		util.Log.Info("Verify NGINX server")
		cmd := fmt.Sprintf(`curl -sS http://my-nginx.bookinfo.svc.cluster.local`)
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "Welcome to nginx") {
			t.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
			util.Log.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
		} else {
			util.Log.Infof("Success. Get expected response: %s", msg)
		}
	})
}
