package certificate

import (
	"fmt"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestAlpnFilterDisabledForNonIstioMtls(t *testing.T) {
	test.NewTest(t).Groups(test.Full).Run(func(t test.TestHelper) {

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns.Foo)
			oc.RecreateNamespace(t, meshNamespace)
		})

		t.Log("Deploying SMCP")
		ossm.DeployControlPlane(t)

		// We need the cluster IP of the ingress gateway service to override DNS resolution during the connection test
		ingressGwServIP := oc.DefaultOC.Invoke(t, `oc get service/istio-ingressgateway -o=jsonpath='{.spec.clusterIP}' -n `+meshNamespace)

		t.Log("Deploying nginx app")
		app.InstallAndWaitReady(t, app.Nginx(ns.Foo))
		oc.CreateTLSSecret(t, meshNamespace, "nginx-server-certs", env.GetRootDir()+"/sampleCerts/nginx.example.com/nginx.example.com.key", env.GetRootDir()+"/sampleCerts/nginx.example.com/nginx.example.com.crt")

		templParamSets := []nginxIngressgatewayYamlTmplParams{
			{
				Ns:                ns.Foo,
				Subset:            false,
				PortLevelSettings: false,
			},
			{
				Ns:                ns.Foo,
				Subset:            true,
				PortLevelSettings: false,
			},
			{
				Ns:                ns.Foo,
				Subset:            false,
				PortLevelSettings: true,
			},
			{
				Ns:                ns.Foo,
				Subset:            true,
				PortLevelSettings: true,
			},
		}

		for _, tmplParam := range templParamSets {

			t.NewSubTest(fmt.Sprintf("Testing connection through gateway with parameters: %+v", tmplParam)).Run(func(t test.TestHelper) {

				t.Cleanup(func() {
					oc.DeleteFromTemplate(t, ns.Foo, nginxIngressgatewayYamlTmpl, tmplParam)
				})

				t.Log(fmt.Sprintf("Applying gateway, virtual service, and destination rule for the nginx app using parameters: %+v", tmplParam))
				oc.ApplyTemplate(t, ns.Foo, nginxIngressgatewayYamlTmpl, tmplParam)

				retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(15).DelayBetweenAttempts(time.Second), func(t test.TestHelper) {

					oc.Exec(t,
						pod.MatchingSelectorFirst("app=istio-ingressgateway,istio=ingressgateway", meshNamespace),
						"istio-proxy",
						// We need to connect to the ingress gateway as if its hostname was 'nginx.example.com'
						fmt.Sprintf(`curl -s -o /dev/null -w '%%{http_code}' https://nginx.example.com/index.html --resolve nginx.example.com:443:%s --insecure`, ingressGwServIP),
						assert.OutputContains("200", "Connection succeeded.", "Connection failed."))
				})
			})
		}
	})
}

type nginxIngressgatewayYamlTmplParams struct {
	Ns                string
	Subset            bool
	PortLevelSettings bool
}

var nginxIngressgatewayYamlTmpl = `
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: my-nginx-gateway
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 443
      name: https
      protocol: HTTPS
    hosts:
    - nginx.example.com
    tls:
      mode: SIMPLE
      credentialName: nginx-server-certs
---
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: my-nginx
spec:
  hosts:
  - nginx.example.com
  gateways:
  - my-nginx-gateway
  http:
  - match:
    - port: 443
    route:
    - destination:
        host: my-nginx.{{ .Ns }}.svc.cluster.local
{{- if .Subset }}
        subset: v1
{{- end }}
        port:
          number: 443
---
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: my-nginx-originate-simple-tls
spec:
  host: my-nginx.{{ .Ns }}.svc.cluster.local
{{- if .Subset }}
  subsets:
  - name: v1
    trafficPolicy:
  {{- if .PortLevelSettings }}
      portLevelSettings:
      - port:
          number: 443
        tls:
          mode: SIMPLE
          sni: nginx.example.com
  {{- else }}
      tls:
        mode: SIMPLE
        sni: nginx.example.com
  {{- end }}
{{- else }}
  trafficPolicy:
  {{- if .PortLevelSettings }}
    portLevelSettings:
    - port:
        number: 443
      tls:
        mode: SIMPLE
        sni: nginx.example.com
  {{- else }}
    tls:
      mode: SIMPLE
      sni: nginx.example.com
  {{- end }}
{{- end }}
`
