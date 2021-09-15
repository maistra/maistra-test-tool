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
	util.Log.Info("Create SMCP v2.1 in istio-system")
	util.ShellMuteOutputError(`oc new-project istio-system`)
	util.KubeApply("istio-system", smcpV21)
	util.KubeApply("istio-system", smmr)
	util.Log.Info("Waiting for mesh installation to complete")
	util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 300s`, "istio-system")
}

func TestSMCPInstall(t *testing.T) {
	defer installDefaultSMCP21()

	t.Run("smcp_test_install_2.1", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Create SMCP v2.1 in istio-system")
		util.ShellMuteOutputError(`oc new-project istio-system`)
		util.KubeApply("istio-system", smcpV21)
		util.KubeApply("istio-system", smmr)
		util.Log.Info("Waiting for mesh installation to complete")
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 300s`, "istio-system")

		util.Log.Info("Verify SMCP status and pods")
		msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, "istio-system", "basic")
		if !strings.Contains(msg, "ComponentsReady") {
			util.Log.Error("SMCP not Ready")
			t.Error("SMCP not Ready")
		}
		util.Shell(`oc get -n %s pods`, "istio-system")
	})

	t.Run("smcp_test_uninstall_2.1", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Delete SMCP v2.1 in istio-system")
		util.KubeDelete("istio-system", smmr)
		util.KubeDelete("istio-system", smcpV21)
		time.Sleep(time.Duration(40) * time.Second)
	})

	t.Run("smcp_test_install_2.0", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Create SMCP v2.0 in namespace istio-system")
		util.ShellMuteOutputError(`oc new-project istio-system`)
		util.KubeApply("istio-system", smcpV20)
		util.KubeApply("istio-system", smmr)
		util.Log.Info("Waiting for mesh installation to complete")
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 300s`, "istio-system")

		util.Log.Info("Verify SMCP status and pods")
		msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, "istio-system", "basic")
		if !strings.Contains(msg, "ComponentsReady") {
			util.Log.Error("SMCP not Ready")
			t.Error("SMCP not Ready")
		}
		util.Shell(`oc get -n %s pods`, "istio-system")
	})

	t.Run("smcp_test_uninstall_2.0", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Delete SMCP v2.0 in istio-system")
		util.KubeDelete("istio-system", smmr)
		util.KubeDelete("istio-system", smcpV20)
		time.Sleep(time.Duration(40) * time.Second)
	})

	t.Run("smcp_test_install_1.1", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Create SMCP v1.1 in namespace istio-system")
		util.ShellMuteOutputError(`oc new-project istio-system`)
		util.KubeApply("istio-system", smcpV11)
		util.KubeApply("istio-system", smmr)
		util.Log.Info("Waiting for mesh installation to complete")
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 300s`, "istio-system")

		util.Log.Info("Verify SMCP status and pods")
		msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, "istio-system", "basic")
		if !strings.Contains(msg, "ComponentsReady") {
			util.Log.Error("SMCP not Ready")
			t.Error("SMCP not Ready")
		}
		util.Shell(`oc get -n %s pods`, "istio-system")
	})

	t.Run("smcp_test_uninstall_1.1", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Delete SMCP v1,1 in istio-system")
		util.KubeDelete("istio-system", smmr)
		util.KubeDelete("istio-system", smcpV11)
		time.Sleep(time.Duration(40) * time.Second)
	})

	t.Run("smcp_test_upgrade_2.0_to_2.1", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Create SMCP v2.0 in namespace istio-system")
		util.ShellMuteOutputError(`oc new-project istio-system`)
		util.KubeApply("istio-system", smcpV20)
		util.KubeApply("istio-system", smmr)
		util.Log.Info("Waiting for mesh installation to complete")
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 300s`, "istio-system")

		util.Log.Info("Verify SMCP status and pods")
		util.Shell(`oc get -n %s smcp/%s -o wide`, "istio-system", "basic")
		util.Shell(`oc get -n %s pods`, "istio-system")

		util.Log.Info("Upgrade SMCP to v2.1 in istio-system")
		util.KubeApply("istio-system", smcpV21)
		util.Log.Info("Waiting for mesh installation to complete")
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 300s`, "istio-system")

		util.Log.Info("Verify SMCP status and pods")
		msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, "istio-system", "basic")
		if !strings.Contains(msg, "ComponentsReady") {
			util.Log.Error("SMCP not Ready")
			t.Error("SMCP not Ready")
		}
		util.Shell(`oc get -n %s pods`, "istio-system")
	})
}
