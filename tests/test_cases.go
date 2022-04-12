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

	// Keep pkg/ossm at the beginning of this import list. It initializes a default SMCP.
	"github.com/maistra/maistra-test-tool/pkg/ossm"

	"github.com/maistra/maistra-test-tool/pkg/tasks/security/authentication"
	authorizaton "github.com/maistra/maistra-test-tool/pkg/tasks/security/authorization"
	"github.com/maistra/maistra-test-tool/pkg/tasks/security/certificate"
	"github.com/maistra/maistra-test-tool/pkg/tasks/traffic"
	"github.com/maistra/maistra-test-tool/pkg/tasks/traffic/egress"
	"github.com/maistra/maistra-test-tool/pkg/tasks/traffic/ingress"

	"github.com/maistra/maistra-test-tool/pkg/federation"
)

var t = &testing.T{}

var smokeTests = []testing.InternalTest{
	testing.InternalTest{
		Name: "A1",
		F:    ossm.TestSMCPInstall,
	},
	testing.InternalTest{
		Name: "A2",
		F:    ossm.TestBookinfo,
	},
	testing.InternalTest{
		Name: "T1",
		F:    traffic.TestRequestRouting,
	},
}

var armCases = []testing.InternalTest{
	testing.InternalTest{
		Name: "A1",
		F:    ossm.TestSMCPInstall,
	},
	testing.InternalTest{
		Name: "A2",
		F:    ossm.TestBookinfo,
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
		F:    traffic.TestRequestTimeouts,
	},
	testing.InternalTest{
		Name: "T5",
		F:    certificate.TestExternalCert,
	},
	testing.InternalTest{
		Name: "T6",
		F:    authorizaton.TestAuthorHTTP,
	},
	testing.InternalTest{
		Name: "T7",
		F:    ossm.TestTLSVersionSMCP,
	},
	testing.InternalTest{
		Name: "T8",
		F:    federation.TestSingleClusterFed,
	},
}

var testCases = []testing.InternalTest{
	testing.InternalTest{
		Name: "A1",
		F:    ossm.TestSMCPInstall,
	},
	testing.InternalTest{
		Name: "A2",
		F:    ossm.TestBookinfo,
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
		F:    ingress.TestIngressGateways,
	},
	testing.InternalTest{
		Name: "T9",
		F:    ingress.TestSecureGateways,
	},
	testing.InternalTest{
		Name: "T10",
		F:    ingress.TestIngressWithoutTLS,
	},
	testing.InternalTest{
		Name: "T11",
		F:    egress.TestAccessExternalServices,
	},
	testing.InternalTest{
		Name: "T12",
		F:    egress.TestEgressTLSOrigination,
	},
	testing.InternalTest{
		Name: "T13",
		F:    egress.TestEgressGateways,
	},
	testing.InternalTest{
		Name: "T15",
		F:    egress.TestTLSOriginationFileMount,
	},
	testing.InternalTest{
		Name: "T16",
		F:    egress.TestEgressWildcard,
	},
	testing.InternalTest{
		Name: "T17",
		F:    certificate.TestExternalCert,
	},
	testing.InternalTest{
		Name: "T18",
		F:    authentication.TestAuthPolicy,
	},
	testing.InternalTest{
		Name: "T19",
		F:    authentication.TestMigration,
	},
	testing.InternalTest{
		Name: "T20",
		F:    authorizaton.TestAuthorHTTP,
	},
	testing.InternalTest{
		Name: "T21",
		F:    authorizaton.TestAuthorTCP,
	},
	testing.InternalTest{
		Name: "T22",
		F:    authorizaton.TestAuthorJWT,
	},
	testing.InternalTest{
		Name: "T23",
		F:    authorizaton.TestAuthorDeny,
	},
	testing.InternalTest{
		Name: "T24",
		F:    authorizaton.TestTrustDomainMigration,
	},

	testing.InternalTest{
		Name: "T25",
		F:    ossm.TestExtensionInstall,
	},
	testing.InternalTest{
		Name: "T26",
		F:    ossm.TestTLSVersionSMCP,
	},
	testing.InternalTest{
		Name: "T27",
		F:    ossm.TestSSL,
	},
	//testing.InternalTest{
	//	Name: "T28",
	//	F:    ossm.TestRateLimiting,
	//},
	//testing.InternalTest{
	//	Name: "T29",
	//	F:    ossm.TestSMCPAnnotations,
	//},
	testing.InternalTest{
		Name: "T30",
		F:    ossm.TestMustGather,
	},
	testing.InternalTest{
		Name: "T31",
		F:    federation.TestSingleClusterFed,
	},
	testing.InternalTest{
		Name: "T32",
		F:    federation.TestSingleClusterFedDiffCert,
	},
	testing.InternalTest{
		Name: "T33",
		F:    ossm.TestInitContainer,
	},
	testing.InternalTest{
		Name: "T34",
		F:    ossm.TestSMCPAddons,
	},
}
