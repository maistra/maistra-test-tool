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

package ingress

import (
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	httpbinSampleServerCertKey        = env.GetRootDir() + "/sampleCerts/httpbin.example.com/httpbin.example.com.key"
	httpbinSampleServerCert           = env.GetRootDir() + "/sampleCerts/httpbin.example.com/httpbin.example.com.crt"
	httpbinSampleCACert               = env.GetRootDir() + "/sampleCerts/httpbin.example.com/example.com.crt"
	httpbinSampleCACrl                = env.GetRootDir() + "/sampleCerts/httpbin.example.com/example.com.crl"
	httpbinSampleClientCert           = env.GetRootDir() + "/sampleCerts/httpbin.example.com/httpbin-client.example.com.crt"
	httpbinSampleClientCertKey        = env.GetRootDir() + "/sampleCerts/httpbin.example.com/httpbin-client.example.com.key"
	httpbinSampleClientRevokedCert    = env.GetRootDir() + "/sampleCerts/httpbin.example.com/httpbin-client-revoked.example.com.crt"
	httpbinSampleClientRevokedCertKey = env.GetRootDir() + "/sampleCerts/httpbin.example.com/httpbin-client-revoked.example.com.key"

	helloworldServerCertKey = env.GetRootDir() + "/sampleCerts/helloworldv1/helloworld-v1.example.com.key"
	helloworldServerCert    = env.GetRootDir() + "/sampleCerts/helloworldv1/helloworld-v1.example.com.crt"

	nginxServerCACert = env.GetRootDir() + "/sampleCerts/nginx.example.com/example.com.crt"
)

var (
	// OCP4.x
	meshNamespace = env.GetDefaultMeshNamespace()
)

func createRoute(t test.TestHelper, ns string, host, targetPort, serviceName string) {
	createRouteWithTLS(t, ns, host, targetPort, serviceName, "")
}

func createRouteWithTLS(t test.TestHelper, ns string, host, targetPort, serviceName string, tlsTermination string) {
	t.LogStepf("Create OpenShift Route for host %s to %s port %s", host, serviceName, targetPort)
	t.Log("NOTE: This is necessary in OSSM 2.4+, because IOR is disabled by default")
	data := map[string]string{
		"host":           host,
		"targetPort":     targetPort,
		"serviceName":    serviceName,
		"tlsTermination": tlsTermination,
	}
	oc.ApplyTemplate(t, ns, routeTemplate, data)
	t.Cleanup(func() {
		oc.DeleteFromTemplate(t, ns, routeTemplate, data)
	})
}

const routeTemplate = `
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: {{ .host }}
spec:
  host: {{ .host }}
  port:
    targetPort: {{ .targetPort }}
{{ if .tlsTermination }}
  tls:
    termination: {{ .tlsTermination }}
{{ end }}
  to:
    kind: Service
    name: {{ .serviceName }}
    weight: 100
  wildcardPolicy: None
`
