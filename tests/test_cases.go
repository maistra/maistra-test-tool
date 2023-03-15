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

var smoke = []testing.InternalTest{
	{Name: "A1", F: ossm.TestSMCPInstall},
	{Name: "A2", F: ossm.TestBookinfo},
	{Name: "T1", F: traffic.TestRequestRouting},
}

var arm = []testing.InternalTest{
	{Name: "A1", F: ossm.TestSMCPInstall},
	{Name: "A2", F: ossm.TestBookinfo},

	{Name: "T1", F: traffic.TestRequestRouting},
	{Name: "T2", F: traffic.TestFaultInjection},
	{Name: "T3", F: traffic.TestTrafficShifting},
	{Name: "T4", F: traffic.TestRequestTimeouts},
	{Name: "T5", F: certificate.TestExternalCert},
	{Name: "T6", F: authorizaton.TestAuthorHTTP},
	{Name: "T7", F: ossm.TestTLSVersionSMCP},
	{Name: "T8", F: federation.TestSingleClusterFed},
}

var full = []testing.InternalTest{
	{Name: "A1", F: ossm.TestSMCPInstall},
	{Name: "A2", F: ossm.TestBookinfo},

	{Name: "T1", F: traffic.TestRequestRouting},
	{Name: "T2", F: traffic.TestFaultInjection},
	{Name: "T3", F: traffic.TestTrafficShifting},
	{Name: "T4", F: traffic.TestTCPShifting},
	{Name: "T5", F: traffic.TestRequestTimeouts},
	{Name: "T6", F: traffic.TestCircuitBreaking},
	{Name: "T7", F: traffic.TestMirroring},
	{Name: "T8", F: ingress.TestIngressGateways},
	{Name: "T9", F: ingress.TestSecureGateways},
	{Name: "T10", F: ingress.TestIngressWithoutTLS},
	{Name: "T11", F: egress.TestAccessExternalServices},
	{Name: "T12", F: egress.TestEgressTLSOrigination},
	{Name: "T13", F: egress.TestEgressGateways},
	{Name: "T14", F: egress.TestTLSOriginationFileMount},
	{Name: "T15", F: egress.TestTLSOriginationSDS},
	{Name: "T16", F: egress.TestEgressWildcard},
	{Name: "T17", F: certificate.TestExternalCert},
	{Name: "T18", F: authentication.TestAuthPolicy},
	{Name: "T19", F: authentication.TestMigration},
	{Name: "T20", F: authorizaton.TestAuthorHTTP},
	{Name: "T21", F: authorizaton.TestAuthorTCP},
	{Name: "T22", F: authorizaton.TestAuthorJWT},
	{Name: "T23", F: authorizaton.TestAuthorDeny},
	{Name: "T24", F: authorizaton.TestTrustDomainMigration},
	// placeholder for T25 TestWasmPlugin
	{Name: "T26", F: ossm.TestTLSVersionSMCP},
	{Name: "T27", F: ossm.TestSSL},
	{Name: "T28", F: ossm.TestRateLimiting},
	{Name: "T29", F: ossm.TestSMCPAnnotations},
	{Name: "T30", F: ossm.TestMustGather},
	{Name: "T31", F: federation.TestSingleClusterFed},
	{Name: "T32", F: federation.TestSingleClusterFedDiffCert},
	{Name: "T33", F: ossm.TestInitContainer},
	{Name: "T34", F: ossm.TestSMCPAddons},
	{Name: "T35", F: ossm.TestIstioPodProbesFails},
	{Name: "T36", F: ossm.TestSMCPMutiple},
}
var interop = []testing.InternalTest{
	{Name: "A1", F: ossm.TestSMCPInstall},
	{Name: "A2", F: ossm.TestBookinfo},

	{Name: "T1", F: traffic.TestRequestRouting},
	{Name: "T2", F: traffic.TestFaultInjection},
	{Name: "T3", F: traffic.TestTrafficShifting},
	{Name: "T4", F: traffic.TestTCPShifting},
	{Name: "T5", F: traffic.TestRequestTimeouts},
	{Name: "T6", F: traffic.TestCircuitBreaking},
	{Name: "T7", F: traffic.TestMirroring},
	{Name: "T8", F: ingress.TestIngressGateways},
	{Name: "T9", F: ingress.TestSecureGateways},
	{Name: "T10", F: ingress.TestIngressWithoutTLS},
	{Name: "T11", F: egress.TestAccessExternalServices},
	{Name: "T12", F: egress.TestEgressTLSOrigination},
	{Name: "T13", F: egress.TestEgressGateways},
	{Name: "T14", F: egress.TestTLSOriginationFileMount},
	{Name: "T15", F: egress.TestTLSOriginationSDS},
	{Name: "T16", F: egress.TestEgressWildcard},
	{Name: "T17", F: certificate.TestExternalCert},
	{Name: "T18", F: authentication.TestAuthPolicy},
	{Name: "T19", F: authentication.TestMigration},
	{Name: "T20", F: authorizaton.TestAuthorHTTP},
	{Name: "T21", F: authorizaton.TestAuthorTCP},
	{Name: "T22", F: authorizaton.TestAuthorJWT},
	{Name: "T23", F: authorizaton.TestAuthorDeny},
	{Name: "T24", F: authorizaton.TestTrustDomainMigration},
	{Name: "T26", F: ossm.TestTLSVersionSMCP},
	{Name: "T27", F: ossm.TestSSL},
}
