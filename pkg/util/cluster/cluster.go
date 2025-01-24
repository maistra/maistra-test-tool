// Copyright 2025 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"encoding/json"

	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

// SupportsIPv6 detects if the cluster supports ipv6 by looking for the kubernetes Service,
// which should always be there, and inspecting ipFamilies.
func SupportsIPv6(t test.TestHelper) bool {
	t.T().Helper()
	var ipFamilies []string
	ipFamResp := oc.GetJson(t, ns.Default, "Service", "kubernetes", "{.spec.ipFamilies}")
	if err := json.Unmarshal([]byte(ipFamResp), &ipFamilies); err != nil {
		t.Fatalf("Unable to marshal ip family resp: %s", err)
	}

	for _, ipFamily := range ipFamilies {
		if ipFamily == "IPv6" {
			return true
		}
	}

	return false
}
