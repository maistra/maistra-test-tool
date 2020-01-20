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

package main

import (
	"testing"
)

var t = &testing.T{}

var tests = []testing.InternalTest{
	testing.InternalTest {
		Name: "Request_Routing",
		F: TestRequestRouting,
	},
	testing.InternalTest {
		Name: "Fault_Injection",
		F: TestFaultInjection,
	},
	testing.InternalTest {
		Name: "Traffic_Shifting",
		F: TestTrafficShifting,
	},
	testing.InternalTest {
		Name: "Request_Timeouts",
		F: TestRequestTimeouts,
	},
	testing.InternalTest {
		Name: "Circuit_Breaking",
		F: TestCircuitBreaking,
	},
	testing.InternalTest {
		Name: "Mirroring",
		F: TestMirroring,
	},
}

func matchString(a, b string) (bool, error) {
	return a == b, nil
}

func TestMain(m *testing.M) {

	testing.Main(matchString, tests, nil, nil)

}
