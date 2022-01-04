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

func TestSMCPAddons(t *testing.T) {

	t.Run("smcp_test_addons_3scale", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Enable 3scale in a CR. Expected validation error.")
		util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{"spec":{"addons":{"3scale":{"enabled":true}}}}'`, meshNamespace, smcpName)
		time.Sleep(time.Duration(20) * time.Second)

		util.Log.Info("Verify SMCP status")
		msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
		if strings.Contains(msg, "ReconcileError") {
			util.Log.Errorf("SMCP not Ready: %s", msg)
			t.Error("SMCP not Ready")
		}
		util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{"spec":{"addons":{"3scale":{"enabled":false}}}}'`, meshNamespace, smcpName)
		util.Shell(`oc -n %s wait --for condition=Ready smcp/%s --timeout 180s`, meshNamespace, smcpName)
	})
}
