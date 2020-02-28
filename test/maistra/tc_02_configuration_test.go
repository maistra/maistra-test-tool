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
	"maistra/util"
)


func Test02(t *testing.T) {

	defer func() {
		// recover from panic if one occurred. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	log.Infof("# Prepare testing configurations")

	// create namespaces
	log.Infof("Create bookinfo, foo, bar and legacy ns")
	util.ShellSilent("oc new-project bookinfo")
	util.ShellSilent("oc new-project foo")
	util.ShellSilent("oc new-project bar")
	util.ShellSilent("oc new-project legacy")
	time.Sleep(time.Duration(10) * time.Second)

	// update smmr
	util.KubeApplyContents(meshNamespace, smmrDefault, kubeconfigFile)
	time.Sleep(time.Duration(10) * time.Second)
	
	// update mtls to false
	log.Info("Update SMCP mtls to true")
	util.ShellMuteOutput("oc patch -n %s smcp/basic-install --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":false,\"mtls\":{\"enabled\":false}}}}}'", meshNamespace)

	// patch smcp ingressgateway
	//util.ShellMuteOutput("")
	//time.Sleep(time.Duration(10) * time.Second)

	// update smcp namespace

	// add anyuid
	log.Infof("Add anyuid scc")
	util.ShellSilent("oc adm policy add-scc-to-user anyuid -z bookinfo-productpage -n bookinfo")
	util.ShellSilent("oc adm policy add-scc-to-user anyuid -z bookinfo-reviews -n bookinfo")
	util.ShellSilent("oc adm policy add-scc-to-user anyuid -z bookinfo-ratings-v2 -n bookinfo")
	util.ShellSilent("oc adm policy add-scc-to-user anyuid -z default -n bookinfo")
	time.Sleep(time.Duration(10) * time.Second)
	

}