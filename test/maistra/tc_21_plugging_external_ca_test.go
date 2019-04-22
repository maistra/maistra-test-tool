// Copyright 2019 Red Hat, Inc.
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

package maistra

import (
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"gopkg.in/yaml.v2"
	"maistra/util"
)

func cleanup21(namespace string, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.Shell("kubectl delete secret cacerts -n istio-system")
	util.Shell("kubectl rollout undo deployment -n istio-system istio-citadel")
	util.ShellMuteOutput("rm -f /tmp/istio-citadel-new.yaml")
	cleanBookinfo(namespace, kubeconfig)
	util.ShellMuteOutput("kubectl delete meshpolicy default")
	log.Info("Waiting... Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}


func updateCitadelDeployment() {
	var b  yaml.TypeError	
	log.Infof("%v", b)

}


func Test21(t *testing.T) {

	defer cleanup21(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	log.Infof("# TC_21 Plugging in External CA Key and Certificate")
	log.Info("Enable mTLS")
	util.Inspect(util.KubeApplyContents("", meshPolicy, kubeconfigFile), "failed to apply MeshPolicy", "", t)
	log.Info("Waiting... Sleep 5 seconds...")
	time.Sleep(time.Duration(5) * time.Second)	

	log.Info("Create secret")
	_, err := util.ShellMuteOutput("kubectl create secret generic %s -n %s --from-file %s --from-file %s --from-file %s --from-file %s --kubeconfig=%s", 
								"cacerts",
								"istio-system",
								caCert,
								caCertKey,
								caRootCert,
								caCertChain,
								kubeconfigFile)
	if err != nil {
		log.Infof("Failed to create secret %s\n", "cacerts")
		t.Errorf("Failed to create secret %s\n", "cacerts")
	}
	log.Infof("Secret %s created\n", "cacerts")
	time.Sleep(time.Duration(5) * time.Second)


} 
