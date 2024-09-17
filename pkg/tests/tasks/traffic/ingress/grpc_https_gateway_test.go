// Copyright 2024 Red Hat, Inc.
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

package ingress

import (
	_ "embed"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/grpc_https_gateway.yaml
	grpcurlTLSGatewayHTTPS string

	//go:embed yaml/grpc_echo_server.yaml
	grpcEchoServerTemplate string

	grpcSampleCertKey = env.GetRootDir() + "/sampleCerts/grpc.example.com/grpc.example.com.key"
	grpcSampleCert    = env.GetRootDir() + "/sampleCerts/grpc.example.com/grpc.example.com.crt"
)

func TestExposeGrpcWithHttpsGateway(t *testing.T) {
	test.NewTest(t).Id("T44").Groups(test.Full, test.InterOp, test.ARM).Run(func(t test.TestHelper) {

		t.Log("This test verifies tls decapsulation of grpc messages in gateway.")

		t.Cleanup(func() {
			app.Uninstall(t, app.GrpCurl(ns.Default))
			oc.DeleteNamespace(t, ns.EchoGrpc)
			oc.RecreateNamespace(t, meshNamespace)
			oc.DeleteSecret(t, meshNamespace, "grpc-credential")
		})

		t.LogStep("Create echo-grpc project")
		oc.CreateNamespace(t, ns.EchoGrpc)

		t.LogStep("Deploy Control Plane")
		ossm.DeployControlPlane(t)

		t.LogStep("Update SMMR to include EchoGrpc Namespaces")
		oc.ApplyString(t, meshNamespace, ossm.AppendDefaultSMMR(ns.EchoGrpc))
		oc.WaitSMMRReady(t, meshNamespace)

		t.LogStep("Create Echo Grpc Server Pods")
		oc.ApplyTemplate(t, ns.EchoGrpc, grpcEchoServerTemplate, nil)

		t.LogStep("Create TLS secrets")
		oc.CreateTLSSecret(t, meshNamespace, "grpc-credential", grpcSampleCertKey, grpcSampleCert)

		t.LogStep("Configure a TLS ingress gateway for a single host")
		oc.ApplyString(t, meshNamespace, grpcurlTLSGatewayHTTPS)

		t.LogStep("Install grpcurl image")
		app.Install(t, app.GrpCurl(ns.Default))

		retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(20), func(t test.TestHelper) {
			oc.LogsFromPods(t,
				ns.Default,
				"app=grpcurl",
				assert.OutputContains("EchoTestService", "rpc command worked successfully", "rpc error"))
		})

	})
}
