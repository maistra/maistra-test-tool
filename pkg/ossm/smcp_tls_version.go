// Copyright 2020 Red Hat, Inc.
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
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupTestTLSVersionSMCP() {
	util.Log.Info("Cleanup ...")
	util.Shell(`kubectl patch -n %s smcp/basic --type=json -p='[{"op": "remove", "path": "/spec/security/controlPlane/tls"}]'`, "istio-system")
	util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 180s`, "istio-system")
}

func TestTLSVersionSMCP(t *testing.T) {
	defer cleanupTestTLSVersionSMCP()

	t.Run("Operator_test_smcp_global_tls_minVersion_TLSv1_0", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Update SMCP spec.security.controlPlane.tls.minProtocolVersion: TLSv1_0")
		_, err := util.Shell(`kubectl patch -n %s smcp/basic --type merge -p '{"spec":{"security":{"controlPlane":{"tls":{"minProtocolVersion":"TLSv1_0"}}}}}'`, "istio-system")
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 180s`, "istio-system")
		if err != nil {
			t.Errorf("Failed to update SMCP with tls.maxProtocolVersion: TLSv1_0")
		}
	})

	t.Run("Operator_test_smcp_global_tls_minVersion_TLSv1_1", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Update SMCP spec.security.controlPlane.tls.minProtocolVersion: TLSv1_1")
		_, err := util.Shell(`kubectl patch -n %s smcp/basic --type merge -p '{"spec":{"security":{"controlPlane":{"tls":{"minProtocolVersion":"TLSv1_1"}}}}}'`, "istio-system")
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 180s`, "istio-system")
		if err != nil {
			t.Errorf("Failed to update SMCP with tls.maxProtocolVersion: TLSv1_1")
		}
	})

	t.Run("Operator_test_smcp_global_tls_maxVersion_TLSv1_3", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Update SMCP spec.security.controlPlane.tls.minProtocolVersion: TLSv1_3")
		_, err := util.Shell(`kubectl patch -n %s smcp/basic --type merge -p '{"spec":{"security":{"controlPlane":{"tls":{"maxProtocolVersion":"TLSv1_3"}}}}}'`, "istio-system")
		util.Shell(`oc wait --for condition=Ready -n %s smmr/default --timeout 180s`, "istio-system")
		if err != nil {
			t.Errorf("Failed to update SMCP with tls.maxProtocolVersion: TLSv1_3")
		}
	})
}
