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
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tasks/traffic"
	"github.com/maistra/maistra-test-tool/pkg/tasks/traffic/egress"

	"github.com/maistra/maistra-test-tool/pkg/ossm"
)

var t = &testing.T{}
var testCases = []testing.InternalTest{
	testing.InternalTest{
		Name: "A1",
		F:    ossm.TestSMCPInstall,
	},

	testing.InternalTest{
		Name: "T1",
		F:    traffic.TestRequestRouting,
	},
	testing.InternalTest{
		Name: "T2",
		F:    traffic.TestFaultInjection,
	},
	testing.InternalTest{
		Name: "T3",
		F:    traffic.TestTrafficShifting,
	},
	testing.InternalTest{
		Name: "T4",
		F:    traffic.TestTCPShifting,
	},
	testing.InternalTest{
		Name: "T5",
		F:    traffic.TestRequestTimeouts,
	},
	testing.InternalTest{
		Name: "T6",
		F:    traffic.TestCircuitBreaking,
	},
	testing.InternalTest{
		Name: "T7",
		F:    traffic.TestMirroring,
	},
	testing.InternalTest{
		Name: "T8",
		F:    egress.TestAccessExternalServices,
	},
	testing.InternalTest{
		Name: "T9",
		F:    egress.TestEgressGateways,
	},
	testing.InternalTest{
		Name: "T10",
		F:    egress.TestTLSOriginationFileMount,
	},
	testing.InternalTest{
		Name: "T11",
		F:    egress.TestEgressWildcard,
	},
	testing.InternalTest{
		Name: "T12",
		F:    ossm.TestTLSVersionSMCP,
	},
}
