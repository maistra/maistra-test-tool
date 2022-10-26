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

package examples

import (
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
)

type Nginx struct {
	Namespace string `json:"namespace,omitempty"`
}

func (n *Nginx) Install(config string) {
	util.Log.Info("Create Secret")
	util.CreateTLSSecret("nginx-server-certs", n.Namespace, nginxServerCertKey, nginxServerCert)
	util.Shell(`kubectl create -n %s secret generic nginx-ca-certs --from-file=%s`, n.Namespace, nginxServerCACert)

	util.Log.Info("Create ConfigMap")
	util.Shell(`kubectl create configmap nginx-configmap --from-file=nginx.conf=%s -n %s`, config, n.Namespace)
	time.Sleep(time.Duration(5) * time.Second)

	util.Log.Info("Deploy Nginx")
	util.KubeApply(n.Namespace, nginxYaml)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(n.Namespace, "run=my-nginx")
	time.Sleep(time.Duration(10) * time.Second)
}

// Install_mTLS deploys a nginx server with mtls config in mesh-external namespace
func (n *Nginx) Install_mTLS(config string) {
	util.Log.Info("Create Secret")
	util.CreateTLSSecret("nginx-server-certs", "mesh-external", meshExtServerCertKey, meshExtServerCert)
	util.Shell(`kubectl create -n %s secret generic nginx-ca-certs --from-file=%s`, "mesh-external", nginxServerCACert)

	util.Log.Info("Create ConfigMap")
	util.Shell(`kubectl create configmap nginx-configmap --from-file=nginx.conf=%s -n %s`, config, "mesh-external")
	time.Sleep(time.Duration(5) * time.Second)

	util.Log.Info("Deploy Nginx")
	util.KubeApply("mesh-external", nginxYaml)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning("mesh-external", "run=my-nginx")
	time.Sleep(time.Duration(10) * time.Second)
}

func (n *Nginx) Uninstall() {
	util.Log.Info("Cleanup Nginx")
	util.KubeDelete(n.Namespace, nginxYaml)
	util.Shell(`kubectl delete configmap nginx-configmap -n %s`, n.Namespace)
	util.Shell(`kubectl delete secret nginx-server-certs -n %s`, n.Namespace)
	util.Shell(`kubectl delete secret nginx-ca-certs -n %s`, n.Namespace)
	time.Sleep(time.Duration(10) * time.Second)
}
