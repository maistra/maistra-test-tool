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

package tests

import "testing"

var t = &testing.T{}
var testCases = []testing.InternalTest{
	testing.InternalTest{
		Name: "01",
		F:    TestRequestRouting,
	},
	testing.InternalTest{
		Name: "02",
		F:    TestFaultInjection,
	},
	testing.InternalTest{
		Name: "03",
		F:    TestTrafficShifting,
	},
	testing.InternalTest{
		Name: "04",
		F:    TestTCPShifting,
	},
	testing.InternalTest{
		Name: "05",
		F:    TestRequestTimeouts,
	},
	testing.InternalTest{
		Name: "06",
		F:    TestCircuitBreaking,
	},
	testing.InternalTest{
		Name: "07",
		F:    TestMirroring,
	},
	testing.InternalTest{
		Name: "08",
		F:    TestIngressGateways,
	},
	testing.InternalTest{
		Name: "09",
		F:    TestIngressTLSGateways,
	},
	testing.InternalTest{
		Name: "10",
		F:    TestIngressWithOutTLS,
	},
	testing.InternalTest{
		Name: "11",
		F:    TestAccessExternalServices,
	},
	testing.InternalTest{
		Name: "12",
		F:    TestEgressTLSOrigination,
	},
	testing.InternalTest{
		Name: "13",
		F:    TestEgressGateways,
	},
	testing.InternalTest{
		Name: "14",
		F:    TestEgressGatewaysTLSOrigination,
	},
	testing.InternalTest{
		Name: "15",
		F:    TestEgressWildcards,
	},
	testing.InternalTest{
		Name: "17",
		F:    TestAuthPolicy,
	},
	testing.InternalTest{
		Name: "18",
		F:    TestAuthMTLSMigration,
	},
	testing.InternalTest{
		Name: "19",
		F:    TestAuthorizationHTTP,
	},
	testing.InternalTest{
		Name: "20",
		F:    TestAuthorizationTCP,
	},
	testing.InternalTest{
		Name: "21",
		F:    TestAuthorizationJWT,
	},
	testing.InternalTest{
		Name: "22",
		F:    TestAuthorizationDeny,
	},
	testing.InternalTest{
		Name: "23",
		F:    TestEnablePolicyEnforcement,
	},
	testing.InternalTest{
		Name: "24",
		F:    TestRateLimits,
	},
	testing.InternalTest{
		Name: "25",
		F:    TestControlHeadersRouting,
	},
	testing.InternalTest{
		Name: "26",
		F:    TestDenials,
	},
	testing.InternalTest{
		Name: "27",
		F:    TestTLSVersionSMCP,
	},
	testing.InternalTest{
		Name: "28",
		F:    TestSSL,
	},
	testing.InternalTest{
		Name: "29",
		F:    TestIngressLoad,
	},
	testing.InternalTest{
		Name: "34",
		F:    TestInvalidSMCPFields,
	},
	testing.InternalTest{
		Name: "36",
		F:    TestSMMROVN,
	},
	testing.InternalTest{
		Name: "37",
		F:    TestExcludeOutboundPortsAnnotation,
	},
	testing.InternalTest{
		Name: "38",
		F:    TestPathNormalizationSMCP,
	},
	testing.InternalTest{
		Name: "39",
		F:    TestDuplicateEntryService,
	},
}
