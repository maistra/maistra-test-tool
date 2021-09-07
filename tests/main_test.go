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

package tests

import (
	"os"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util"
)

var (
	SMMR = "../templates/smmr-templates/smmr_default.yaml"
)

func setupNamespaces() {
	util.ShellSilent(`oc new-project bookinfo`)
	util.ShellSilent(`oc new-project foo`)
	util.ShellSilent(`oc new-project bar`)
	util.ShellSilent(`oc new-project legacy`)
	util.ShellSilent(`oc new-project mesh-external`)
	util.ShellSilent(`oc apply -n istio-system -f %s`, SMMR)
	util.ShellSilent(`oc wait --for condition=Ready -n %s smmr/default --timeout 180s`, "istio-system")
}

func matchString(a, b string) (bool, error) {
	return a == b, nil
}

func TestMain(m *testing.M) {
	os.Setenv("GODEBUG", "x509ignoreCN=0")
	setupNamespaces()
	// test runs
	testing.Main(matchString, testCases, nil, nil)
}
