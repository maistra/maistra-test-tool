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
	"fmt"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/shell"

	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestSMCPSecret(t *testing.T) {
	NewTest(t).Id("T52").Groups(Full, Disconnected, ARM, Persistent).Run(func(t TestHelper) {
		t.Log("Verify that secret begins with $2a$, indicating it's been hashed with bcrypt")
		t.Log("Reference: https://issues.redhat.com/browse/OSSM-1094")

		DeployControlPlane(t)

		output := shell.Execute(t, fmt.Sprintf(`oc get secret -n %s htpasswd -o json | jq .data.auth | tr -d \" | base64 -d | sed 's/}.*/}REDACTED\n/'`, meshNamespace))
		str := "$2a$"

		if strings.Contains(output, str) {
			t.LogSuccess(fmt.Sprintf("string '%s' found in response", str))
		} else {
			t.Fatalf("expected to find the string '%s' in the response, but it wasn't found", str)
		}
	})
}
