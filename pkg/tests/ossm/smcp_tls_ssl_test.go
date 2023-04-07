// Copyright 2021 Red Hat, Inc.
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
package ossm

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestTLSVersionSMCP(t *testing.T) {
	NewTest(t).Id("T26").Groups(Full, ARM, InterOp).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		ns := "bookinfo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
			oc.RecreateNamespace(t, meshNamespace)
			oc.ApplyString(t, meshNamespace, smcpName)
			t.LogStep("Enable the Service Mesh Control Plane mTLS to true")
			oc.Patch(t, meshNamespace,
					fmt.Sprintf(`smcp/%s`, smcpName),
				    "merge",
					`{"spec":{"security":{"dataPlane":{"mtls":true},"controlPlane":{"mtls":true}}}}'`,
				)
			t.LogStep("Update SMCP spec.security.controlPlane.tls")
			oc.Patch(t, meshNamespace,
					fmt.Sprint(`smcp/%s`, smcpName),
					"merge",
					`"spec":{"security":{"controlPlane":{"tls"`,
					`"minProtocolVersion":"TLSv1_2"`,
					`"maxProtocolVersion":"TLSv1_2"`,
					`"cipherSuites":["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"]`,
							`"ecdhCurves":["CurveP256", "CurveP384"]`,
						)
		})
	
		t.Log("This test checks if the SMCP updated the tls.maxProtocolVersion to TLSv1_0")
		t.LogStep("Update SMCP spec.security.controlPlane.tls.minProtocolVersion: TLS_v1_0 and sleep")
		app.InstallAndWaitReady(t, app.Bookinfo(ns), app.Sleep(ns))
		
		t.NewSubTest("Operator_test_smcp_global_tls_minVersion_TLSv1_0").Run(func(t TestHelper) {
			t.LogStep("Check to see if the SMCP minProtocolVersion is TLSv1_0")
			retry.UntilSuccess(t, func(t TestHelper) {
				t.Log("Update SMCP spec.security.controlPlane.tls.minProtocolVersion: TLSv1_0")
				oc.Patch(t, ns, "smcp/"+smcpName, "merge", `{"spec":{"security":{"controlPlane":{"tls":{"minProtocolVersion":"TLSv1_0"}}}}}`)
				oc.WaitSMCPReady(t, ns, smcpName)
			})
		})

		t.NewSubTest("Operator_test_smcp_global_tls_minVersion_TLSv1_1").Run(func(t TestHelper) {

			t.LogStep("Check to see if the SMCP minProtocolVersion is TLSv1_1")
			retry.UntilSuccess(t, func(t TestHelper) {
				t.Log("Update SMCP spec.security.controlPlane.tls.minProtocolVersion: TLSv1_1")
				oc.Patch(t, ns, "smcp/"+smcpName, "merge", `{"spec":{"security":{"controlPlane":{"tls":{"minProtocolVersion":"TLSv1_1"}}}}}`)
				oc.WaitSMCPReady(t, ns, smcpName)
			})
		})
		t.NewSubTest("Operator_test_smcp_global_tls_minVersion_TLSv1_3").Run(func(t TestHelper) {

			t.LogStep("Check to see if the SMCP minProtocolVersion is TLSv1_3")
			retry.UntilSuccess(t, func(t TestHelper) {
				t.Log("Update SMCP spec.security.controlPlane.tls.minProtocolVersion: TLSv1_3")
				oc.Patch(t, ns, "smcp/"+smcpName, "merge", `{"spec":{"security":{"controlPlane":{"tls":{"minProtocolVersion":"TLSv1_3"}}}}}`)
				oc.WaitSMCPReady(t, ns, smcpName)
			})
		
		})
	})
}

func TestSSL(t TestHelper) {
	NewTest(t).Id("T27").Groups(Full, InterOp).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		n := "bookinfo"
		t.Cleanup(func() {
			oc.DeleteNamespace(t, ns)
			oc.RecreateNamespace(t, ns)
			oc.ApplyString(t, meshNamespace, smcpName)		
		})

		t.NewSubTest("Testing Operator test for testssl").Run(func(t TestHelper) {
			t.LogStep("Enable the Service Mesh Control Plane mTLS to true")
			retry.UntilSuccess(t, func(t TestHelper) {
				t.Log("The Service Mesh Control Plane mTLS is true")
				oc.Patch(t, meshNamespace,
					fmt.Sprintf(`smcp/%s`, smcpName),
					"merge",
					`{"spec":{"security":{"dataPlane":{"mtls":true},"controlPlane":{"mtls":true}}}}'`,
			
				) 	
			})
		})	
		

		oc.WaitSMCPReady(t, ns, smcpName)
		t.LogStep("Update SMCP spec.security.controlPlane.tls")
		oc.Patch(t, meshNamespace,
				fmt.Sprint(`smcp/%s`, smcpName),
				"merge",
				`"spec":{"security":{"controlPlane":{"tls":{"maxProtocolVersion":"TLSv1_3"}}}}}'`,
				`"minProtocolVersion":"TLSv1_2"`,
				`"maxProtocolVersion":"TLSv1_2"`,
				`"cipherSuites":["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"]`,
				`"ecdhCurves":["CurveP256", "CurveP384"]`,
					)

		oc.Patch(t, ns, "smcp/"+smcpName, "merge" )
		oc.Log("Deploy bookinfo")
		t.LogStep("Install bookinfo with mTLS")
		app.InstallAndWaitReady(t, app.BookinfoWithMTLS(ns))
		oc.Log("Deploy testssl pod")
		oc.ApplyString(t, ns, testSSLDeploymentP)
		oc.ApplyString(t, ns, testSSLDeploymentZ)
		oc.ApplyString(t, ns, testSSLDeployment)
	})
}			   

func assertRequestDenied(t TestHelper, ns string, curlCommand string) {
	retry.UntilSuccess(t, func(t TestHelper) {
	   oc.Exec(t,
		   pod.MatchingSelector("app=testssl", ns),
		   "testssl",
		   curlCommand,
		   assert.OutputContains(
			   "TLSv1.2",
			   "Recieved the TLSv1.2 needed in the testssl.sh results",
			   "Expected to recieve TLSv1.2 in the testssl.sh results, but recieved something different"),
			assert.OutputContains(
			   "ECDHE-RSA-AES128-GCM-SHA256",
			   "Results recieved the correct SHA256",
			   "Results not include: ECDHE-RSA-AES128-GCM-SHA256"),
			assert.OutputContains(
			   "P-256",
			   "Results included: P-256",
			   "Results not include: P-256")),
	})
}
  
		   
		   





   




const (
	   testSSLDeploymentP = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: testssl
spec:
  replicas: 1
  selector:
    matchLabels:
      app: testssl
  template:
    metadata:
      labels:
        app: testssl
    spec:
      containers:
      - name: testssl
        image: quay.io/maistra/testssl:0.0-ibm-p
        imagePullPolicy: Always` 

	   testSSLDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: testssl
spec:
  replicas: 1
    selector:
			matchLabels:
			  app: testssl
		  template:
			metadata:
			  labels:
				app: testssl
			spec:
			  containers:
			  - name: testssl
				image: quay.io/maistra/testssl:latest
				imagePullPolicy: Always`
		

	   testSSLDeploymentZ = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: testssl
spec:
  replicas: 1
  selector:
    matchLabels:
      app: testssl
  template:
    metadata:
      labels:
        app: testssl
    spec:
      containers:
      - name: testssl
        image: quay.io/maistra/testssl:0.0-ibm-z
        imagePullPolicy: Always`
)
