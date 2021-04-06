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

package smcp

import (
	"github.com/maistra/maistra-test-tool/pkg/util/cli"
	logger "github.com/maistra/maistra-test-tool/pkg/util/log"
)

// CRv2, CRv1 CR templates in resources. SMMR template in resources.
const (
	CRv2 = "../../resources/smcp-templates/v2.0/cr_2.0_default.yaml"
	CRv1 = "../../resources/smcp-templates/v1.1/cr_1.1_default.yaml"
	SMMR = "../../resources/smmr_templates/smmr_default.yaml"
)

var log = logger.NewTextLogger()

// ControlPlane contains CP namespace, version and memebers from SMMR
type ControlPlane struct {
	// Name 		string `json:"name,omitempty"`
	Namespace 	string `json:"namespace,omitempty"`
	Version  	string `json:"version,omitempty"`
	Members		[]string
}

func addMembers(cp *ControlPlane) {
	for _, member := range(cp.Members) {
		_, err := cli.Shell(`oc new-project %s`, member)
		if err != nil {
			cli.Shell(`kubectl create ns %s`, member)
		}
	}
	cli.Shell(`kubectl apply -n %s -f %s`, cp.Namespace, SMMR)
}

func create(cp *ControlPlane) {
	log.Info("Create SMCP namespace")
	_, err := cli.Shell(`oc new-project %s`, cp.Namespace)
	if err != nil {
		cli.Shell(`kubectl create ns %s`, cp.Namespace)
	}

	log.Info("Create SMCP")
	if cp.Version == "v2.0" {
		cli.Shell(`kubectl apply -n %s -f %s`, cp.Namespace, CRv2)
	}
	if cp.Version == "v1.1" {
		cli.Shell(`kubectl apply -n %s -f %s`, cp.Namespace, CRv1)
	}
	
	log.Info("Check installation events...")
	
	template := `'{{range .status.conditions}}{{printf "%s=%s, reason=%s, message=%s\n\n" .type .status .reason .message}}{{end}}'`
	cli.Shell(`kubectl get -n %s ServiceMeshControlPlane/basic --template=%s`, cp.Namespace, template)
	

}

func checkStatus(cp *ControlPlane) {

}

func delete(cp *ControlPlane) {
	log.Info("Delete SMMR")
	cli.Shell(`kubectl delete -n %s -f %s`, cp.Namespace, SMMR)

	log.Info("Delete SMCP")
	if cp.Version == "v2.0" {
		cli.Shell(`kubectl delete -n %s -f %s`, cp.Namespace, CRv2)
	}
	if cp.Version == "v1.1" {
		cli.Shell(`kubectl delete -n %s -f %s`, cp.Namespace, CRv1)
	}

}

func getImages(cp *ControlPlane) {
	log.Info("Verify image names")
	cli.Shell(`kubectl get pods -n %s -o jsonpath="{..image}"`)
	log.Info("Verify image IDs")
	cli.Shell(`kubectl get pods -n %s -o jsonpath="{..imageID}"`)
	log.Info("Verify rpm names")
	cli.Shell(`kubectl get pods -n %s -o go-template --template='{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}'`)
}

