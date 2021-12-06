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
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
)

func installDefaultSMCP21() {
	util.Log.Info("Create SMCP v2.1 in ", meshNamespace)
	util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
	util.Shell(`envsubst < %s > %s`, smcpV21_template, smcpV21)
	util.KubeApply(meshNamespace, smcpV21)
	util.KubeApply(meshNamespace, smmr)
	util.Log.Info("Waiting for mesh installation to complete")
	util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 300s`, meshNamespace)
}

func TestSMCPInstall(t *testing.T) {
	defer installDefaultSMCP21()

	t.Run("smcp_test_install_2.1", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Create SMCP v2.1 in ", meshNamespace)
		util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
		util.Shell(`envsubst < %s > %s`, smcpV21_template, smcpV21)
		util.KubeApply(meshNamespace, smcpV21)
		util.KubeApply(meshNamespace, smmr)
		util.Log.Info("Waiting for mesh installation to complete")
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 300s`, meshNamespace)

		util.Log.Info("Verify SMCP status and pods")
		msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
		if !strings.Contains(msg, "ComponentsReady") {
			util.Log.Error("SMCP not Ready")
			t.Error("SMCP not Ready")
		}
		util.Shell(`oc get -n %s pods`, meshNamespace)
	})

	t.Run("smcp_test_uninstall_2.1", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Delete SMCP v2.1 in ", meshNamespace)
		util.KubeDelete(meshNamespace, smmr)
		util.Shell(`envsubst < %s > %s`, smcpV21_template, smcpV21)
		util.KubeDelete(meshNamespace, smcpV21)
		time.Sleep(time.Duration(40) * time.Second)
	})

	t.Run("smcp_test_install_2.0", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Create SMCP v2.0 in namespace ", meshNamespace)
		util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
		util.Shell(`envsubst < %s > %s`, smcpV20_template, smcpV20)
		util.KubeApply(meshNamespace, smcpV20)
		util.KubeApply(meshNamespace, smmr)
		util.Log.Info("Waiting for mesh installation to complete")
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 300s`, meshNamespace)

		util.Log.Info("Verify SMCP status and pods")
		msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
		if !strings.Contains(msg, "ComponentsReady") {
			util.Log.Error("SMCP not Ready")
			t.Error("SMCP not Ready")
		}
		util.Shell(`oc get -n %s pods`, meshNamespace)
	})

	t.Run("smcp_test_uninstall_2.0", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Delete SMCP v2.0 in ", meshNamespace)
		util.KubeDelete(meshNamespace, smmr)
		util.Shell(`envsubst < %s > %s`, smcpV20_template, smcpV20)
		util.KubeDelete(meshNamespace, smcpV20)
		time.Sleep(time.Duration(40) * time.Second)
	})

	t.Run("smcp_test_install_1.1", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Create SMCP v1.1 in namespace ", meshNamespace)
		util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
		util.Shell(`envsubst < %s > %s`, smcpV11_template, smcpV11)
		util.KubeApply(meshNamespace, smcpV11)
		util.KubeApply(meshNamespace, smmr)
		util.Log.Info("Waiting for mesh installation to complete")
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 300s`, meshNamespace)

		util.Log.Info("Verify SMCP status and pods")
		msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
		if !strings.Contains(msg, "ComponentsReady") {
			util.Log.Error("SMCP not Ready")
			t.Error("SMCP not Ready")
		}
		util.Shell(`oc get -n %s pods`, meshNamespace)
	})

	t.Run("smcp_test_uninstall_1.1", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Delete SMCP v1,1 in ", meshNamespace)
		util.KubeDelete(meshNamespace, smmr)
		util.Shell(`envsubst < %s > %s`, smcpV11_template, smcpV11)
		util.KubeDelete(meshNamespace, smcpV11)
		time.Sleep(time.Duration(40) * time.Second)
	})

	t.Run("smcp_test_upgrade_2.0_to_2.1", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Create SMCP v2.0 in namespace ", meshNamespace)
		util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
		util.Shell(`envsubst < %s > %s`, smcpV20_template, smcpV20)
		util.KubeApply(meshNamespace, smcpV20)
		util.KubeApply(meshNamespace, smmr)
		util.Log.Info("Waiting for mesh installation to complete")
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 300s`, meshNamespace)

		util.Log.Info("Verify SMCP status and pods")
		util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
		util.Shell(`oc get -n %s pods`, meshNamespace)

		util.Log.Info("Upgrade SMCP to v2.1 in istio-system")
		util.Shell(`envsubst < %s > %s`, smcpV21_template, smcpV21)
		util.KubeApply(meshNamespace, smcpV21)
		util.Log.Info("Waiting for mesh installation to complete")
		time.Sleep(time.Duration(10) * time.Second)
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 360s`, meshNamespace)

		util.Log.Info("Verify SMCP status and pods")
		msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
		if !strings.Contains(msg, "ComponentsReady") {
			util.Log.Error("SMCP not Ready")
			t.Error("SMCP not Ready")
		}
		util.Shell(`oc get -n %s pods`, meshNamespace)
	})
}
