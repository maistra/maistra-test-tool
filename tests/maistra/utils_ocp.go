// Copyright 2019 Istio Authors
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

// Package dashboard provides testing of the grafana dashboards used in Istio
// to provide mesh monitoring capabilities.

package maistra


import (
	"fmt"
	
	"istio.io/istio/tests/util"
)


// OcLogin runs oc login command to log into the OCP CLI
// the host and token can be found from OCP web console Command Line Tools
func OcLogin(host, token string) error {
	_, err := util.Shell("oc login https://%s:8443 --token=%s", host, token)
	return err
}

func ocCommand(subCommand, namespace, yamlFileName string, kubeconfig string) string {
	if namespace == "" {
		return fmt.Sprintf("oc %s -f %s --kubeconfig=%s", subCommand, yamlFileName, kubeconfig)
	}
	return fmt.Sprintf("oc %s -n %s -f %s --kubeconfig=%s", subCommand, namespace, yamlFileName, kubeconfig)
}

// OcApply oc apply from file
func OcApply(namespace, yamlFileName string, kubeconfig string) error {
	_, err := util.Shell(ocCommand("apply", namespace, yamlFileName, kubeconfig))
	return err
}

// OcDelete kubectl delete from file
func OcDelete(namespace, yamlFileName string, kubeconfig string) error {
	_, err := util.Shell(ocCommand("delete", namespace, yamlFileName, kubeconfig))
	return err
}
