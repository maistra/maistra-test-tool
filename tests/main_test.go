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

/*
 * This package includes an entrypoint of running tests.
 * main_test.go is calling Golang testing.Main framework and it reloads all packages from pkg directory.
 * All test cases are mapped in the test_cases.go file.
 */

package tests

import (
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util"
)

// Create namespaces. All test samples and configurations will be in those namespaces.
func setupNamespaces() {
	util.ShellSilent(`oc new-project bookinfo`)
	util.ShellSilent(`oc new-project foo`)
	util.ShellSilent(`oc new-project bar`)
	util.ShellSilent(`oc new-project legacy`)
	util.ShellSilent(`oc new-project mesh-external`)
}

// this function is used for matching command line argument <test case name>,
// e.g. `go test -run <test case name>` with the names in the test_cases.go file.
func matchString(a, b string) (bool, error) {
	return a == b, nil
}

func TestMain(m *testing.M) {
	setupNamespaces()

	if util.Getenv("SAMPLEARCH", "x86") == "x86" ||
	   util.Getenv("SAMPLEARCH", "x86") == "p" ||
	   util.Getenv("SAMPLEARCH", "x86") == "z" {
		// Search all test cases in a regular run
		testing.Main(matchString, testCases, nil, nil)  // testCases constant is defined in test_cases.go
	} else if util.Getenv("SAMPLEARCH", "x86") == "arm" {
		// Search test cases in an ARM run
		testing.Main(matchString, armCases, nil, nil)  // armCases constant is defined in test_cases.go
	}
}
