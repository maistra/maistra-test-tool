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

package ossm

import (
	"github.com/maistra/maistra-test-tool/pkg/config"
	"github.com/maistra/maistra-test-tool/pkg/util"
)

// CRv2, CRv1 CR templates in resources. SMMR template in resources.
const (
	CRv21 = "../samples/ossm/cr_2.1_default.yaml"
	CRv20 = "../samples/ossm/cr_2.0_default.yaml"
	CRv11 = "../samples/ossm/cr_1.1_default.yaml"
	SMMR = "../samples/ossm/smmr_default.yaml"
)

// ControlPlane contains CP namespace, version and memebers from SMMR
type ControlPlane struct {
	// Name 		string `json:"name,omitempty"`
	Namespace 	string `json:"namespace,omitempty"`
	Version  	string `json:"version,omitempty"`
	Members		[]string
}

func (cp *ControlPlane) Install(CR string) {
	util.Log.Info("Create SMCP namespace")
	config.CreateNamespace(cp.Namespace)

	for _, member := range(cp.Members) {
		config.CreateNamespace(member)
	}

	util.Log.Info("Create SMCP")
	util.Shell(`oc create -n %s -f %s`, cp.Namespace, CR)
	util.Shell(`oc create -n %s -f %s`, cp.Namespace, SMMR)
	
	util.Log.Info("Waiting for mesh installation to complete")
	util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 300s`, cp.Namespace)
}

func (cp *ControlPlane) Uninstall(CR string) {
	util.Log.Info("Uninstall SMCP")
	util.Shell(`oc delete -n %s -f %s`, cp.Namespace, SMMR)
	util.Shell(`sleep 10`)
	util.Shell(`oc delete -n %s -f %s`, cp.Namespace, CR)
	util.Shell(`sleep 40`)
}

func (cp *ControlPlane) CheckStatus() {
	util.Shell(`oc get smcp/basic -n %s -o wide`, cp.Namespace)
}

func (cp *ControlPlane) CheckImages() {
	util.Log.Info("Verify image names")
	util.Shell(`oc get pods -n %s -o jsonpath="{..image}"`, cp.Namespace)
	util.Log.Info("Verify image IDs")
	util.Shell(`oc get pods -n %s -o jsonpath="{..imageID}"`, cp.Namespace)
}
