// Copyright Red Hat, Inc.
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

	"github.com/maistra/maistra-test-tool/pkg/util"
)

var smcpOriginal string

func createProfiles(t *testing.T) {
	if _, err := util.Shell("cat <<EOF | kubectl apply -n %s -f - \n%s \nEOF", "openshift-operators", `{
		"apiVersion": "v1",
			"data": {
				"test1": "apiVersion: maistra.io/v1\nkind: ServiceMeshControlPlane\nmetadata:\n  name: auth-install\nspec:\n  tracing:\n    sampling: 1337\n",
				"test2": "apiVersion: maistra.io/v1\nkind: ServiceMeshControlPlane\nmetadata:\n  name: auth-install\nspec:\n  tracing:\n    sampling: 1338\n"
			},
			"kind": "ConfigMap",
			"metadata": {
				"name": "smcp-templates",
				"namespace": "openshift-operators"
			}
		}`); err != nil {
		t.Fatalf("Failed to create SMCP profiles: %s", err.Error())
	}
}

func removeProfiles(t *testing.T) {
	if _, err := util.Shell("kubectl delete -n %s configmap smcp-templates", "openshift-operators"); err != nil {
		t.Fatalf("Failed to clean up after test: %s", err.Error())
	}
}

func restoreOriginalSMCP(t *testing.T) {
	if err := util.KubeApply("istio-system", smcpV21); err != nil {
		t.Fatalf("Failed to apply SMCP: %s", err.Error())
	}
}

func TestSMCPProfile(t *testing.T) {
	t.Run("values_retrieved_from_profile", func(t *testing.T) {
		defer util.RecoverPanic(t)
		defer restoreOriginalSMCP(t)

		createProfiles(t)
		defer removeProfiles(t)

		util.Log.Info("Test that values can be retrieved from a profile")
		util.Shell(`kubectl -n %s patch --type=json smcp/basic -p='[{"op":"add","path":"/spec/profiles", "value":["default", "test1"]}]'`, "istio-system")

		_, err := util.Shell(`oc wait --for condition=Ready -n %s smcp/basic --timeout 180s`, "istio-system")
		if err != nil {
			t.Fatalf("Failed to set and SMCP with no specified profiles: %s", err.Error())
		}

		msg, _ := util.Shell(`kubectl get smcp -n %s -o yaml | grep sampling -m 1`, "istio-system")
		if !strings.Contains(msg, "test_one") {
			t.Fatalf("Failed to retrieve value from SMCP. Expected %s. Got %s", "test_one", msg)
		}
	})

	t.Run("values_override_smcp_profile", func(t *testing.T) {
		defer util.RecoverPanic(t)
		defer restoreOriginalSMCP(t)

		createProfiles(t)
		defer removeProfiles(t)

		util.Log.Info("Test that using a profile applies the smcp on top of the profile")
		util.Shell(`kubectl -n %s patch --type=json smcp/basic -p='[{"op":"add","path":"/spec/profiles", "value":["default", "test1"]}]'`, "istio-system")

		util.Shell(`kubectl -n %s patch --type=json smcp/basic -p='[{"op":"add","path":"/spec/techPreview", "value":{"sampling":"1339"}}]'`, "istio-system")
		defer util.Shell(`kubectl -n %s patch --type=json smcp/basic -p='[{"op":"add","path":"/spec/techPreview", "value":{"sampling":"test_three"}}]'`, "istio-system")

		_, err := util.Shell(`oc wait --for condition=Ready -n %s smcp/basic --timeout 180s`, "istio-system")
		if err != nil {
			t.Fatalf("Failed to set and SMCP with no specified profiles: %s", err.Error())
		}

		msg, _ := util.Shell(`kubectl get smcp -n %s -o yaml | grep maistra-test-value -m 1`, "istio-system")
		if !strings.Contains(msg, "test_three") {
			t.Fatalf("Failed to retrieve value from SMCP. Expected %s. Got %s", "test_three", msg)
		}
	})

	t.Run("no_profiles_picks_up_defaults", func(t *testing.T) {
		defer util.RecoverPanic(t)
		defer restoreOriginalSMCP(t)

		createProfiles(t)
		defer removeProfiles(t)

		util.Log.Info("Test that specifying no profile causes the SMCP to pick up the defaults (no errors)")
		util.Shell(`kubectl -n %s patch --type=json smcp/basic -p='[{"op":"replace","path":"/spec/profiles", "value":[]}]'`, "istio-system")
		_, err := util.Shell(`oc wait --for condition=Ready -n %s smcp/basic --timeout 180s`, "istio-system")
		if err != nil {
			t.Fatalf("Failed to set and SMCP with no specified profiles: %s", err.Error())
		}

	})

	/*t.Run("one_profile_specified_still_picks_up_defaults", func(t *testing.T) {
		defer util.RecoverPanic(t)
		defer restoreOriginalSMCP(t)
		createProfiles(t)
		defer removeProfiles(t)
		util.Log.Info("Test that specifying only one profile still picks up defaults in addition to the profile changes")
		util.Shell(`kubectl -n %s patch --type=json smcp/basic -p='[{"op":"replace","path":"/spec/profiles", "value":["test1"]}]'`, "istio-system")
		_, err := util.Shell(`oc wait --for condition=Ready -n %s smcp/basic --timeout 180s`, "istio-system")
		if err == nil {
			t.Fatalf("Failed to set an SMCP with only one profile: %s", err.Error())
		}
	})*/

	t.Run("multiple_profiles", func(t *testing.T) {
		defer util.RecoverPanic(t)
		defer restoreOriginalSMCP(t)

		createProfiles(t)
		defer removeProfiles(t)

		util.Log.Info("Test that the last profile overrides the first")
		util.Shell(`kubectl -n %s patch --type=json smcp/basic -p='[{"op":"add","path":"/spec/profiles", "value":["test1", "test2", "default"]}]'`, "istio-system")
		_, err := util.Shell(`oc wait --for condition=Ready -n %s smcp/basic --timeout 180s`, "istio-system")
		if err != nil {
			t.Fatalf("Failed to set an SMCP with no specified profiles: %s", err.Error())
		}

		msg, _ := util.Shell(`kubectl get smcp -n %s -o yaml | grep maistra-test-value -m 1`, "istio-system")
		if !strings.Contains(msg, "test_two") {
			t.Fatalf("Failed to retrieve value from SMCP. Expected %s. Got %s", "test_two", msg)
		}
	})

	t.Run("missing_profile", func(t *testing.T) {
		defer util.RecoverPanic(t)
		defer restoreOriginalSMCP(t)
		createProfiles(t)
		defer removeProfiles(t)

		util.Log.Info("Test that requesting a missing profile causes an error")
		util.Shell(`kubectl -n %s patch --type=json smcp/basic -p='[{"op":"add","path":"/spec/profiles", "value":["missing_smcp", "default"]}]'`, "istio-system")
		_, err := util.Shell(`oc wait --for condition=Reconciled -n %s smcp/basic --timeout 180s`, "istio-system")
		if err == nil {
			t.Fatal("No error returned for missing profile")
		}
	})
}
