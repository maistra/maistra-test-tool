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

package config

import (
	"github.com/maistra/maistra-test-tool/pkg/util"
)

const (
	SMMR = `
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:
  # a list of namespaces that should be joined into the service mesh
  # for example, to add the bookinfo namespace
  - bookinfo
  - foo
  - bar
  - legacy
`
)

// Login: oc login command from an OCP cluster. server is the API server URL
func Login(user, token, server string) {
	util.Shell(`oc login -u %s -p %s --server=%s --insecure-skip-tls-verify=true`, user, token, server)
}

func CreateNamespace(ns string) {
	util.Shell(`oc new-project %s`, ns)
}

// Setup: creates testing namespaces and create a SMMR in a CP namespace 
func Setup(cpns string) {
	// Create namespaces
	util.ShellSilent(`oc new-project bookinfo`)
	util.ShellSilent(`oc new-project foo`)
	util.ShellSilent(`oc new-project bar`)
	util.ShellSilent(`oc new-project legacy`)
	util.ShellSilent(`oc new-project mesh-external`)

	// Add scc for nginx and mongdb examples in bookinfo
	util.ShellSilent(`oc adm policy add-scc-to-user anyuid -z default -n bookinfo`)
	util.ShellSilent(`oc adm policy add-scc-to-user anyuid -z bookinfo-ratings-v2 -n bookinfo`)

	// Apply SMMR in default istio-system ns
	util.KubeApplyContents(cpns, SMMR)
	util.ShellSilent(`sleep 10`)
}