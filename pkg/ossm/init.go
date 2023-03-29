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
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

// Install nightly build operators from quay.io. This is used in Jenkins daily build pipeline.
func installNightlyOperators() {
	util.KubeApply("openshift-operators", jaegerSubYaml)
	util.KubeApply("openshift-operators", kialiSubYaml)
	util.KubeApply("openshift-operators", ossmSubYaml)
	time.Sleep(time.Duration(60) * time.Second)
	util.CheckPodRunning("openshift-operators", "name=istio-operator")
	time.Sleep(time.Duration(30) * time.Second)
}

// Initialize a default SMCP and SMMR
func init() {

	if env.Getenv("NIGHTLY", "false") == "true" {
		installNightlyOperators()
	}

	util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
	util.KubeApplyContents(meshNamespace, util.RunTemplate(smcpV23_template, smcp))
	util.KubeApplyContents(meshNamespace, smmr)
	util.Shell(`oc -n %s wait --for condition=Ready smcp/%s --timeout 180s`, meshNamespace, smcp.Name)
	util.Shell(`oc -n %s wait --for condition=Ready smmr/default --timeout 180s`, meshNamespace)
	if ipv6 == "true" {
		log.Log.Info("Running the test with IPv6 configuration")
	}

}
