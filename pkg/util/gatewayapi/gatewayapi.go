package gatewayapi

import (
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func InstallSupportedVersion(t test.TestHelper, smcp version.Version) {
	shell.Executef(t, "kubectl get crd gateways.gateway.networking.k8s.io &> /dev/null && echo 'Gateway API CRDs already installed' || kubectl apply -k %s", getSupportedVersion(smcp))
}

func UninstallSupportedVersion(t test.TestHelper, smcp version.Version) {
	shell.Executef(t, "kubectl get crd gateways.gateway.networking.k8s.io &> /dev/null && kubectl delete -k %s || echo 'Gateway API CRDs not found, nothing to delete'", getSupportedVersion(smcp))
}

func getSupportedVersion(smcp version.Version) string {
	switch smcp {
	case version.SMCP_2_3:
		return "github.com/kubernetes-sigs/gateway-api/config/crd/experimental?ref=v0.5.1"
	case version.SMCP_2_4:
		return "github.com/kubernetes-sigs/gateway-api/config/crd/experimental?ref=v0.5.1"
	case version.SMCP_2_5:
		return "github.com/kubernetes-sigs/gateway-api/config/crd/experimental?ref=v0.6.2"
	case version.SMCP_2_6:
		return "github.com/kubernetes-sigs/gateway-api/config/crd?ref=v1.0.0"
	default:
		return "github.com/kubernetes-sigs/gateway-api/config/crd?ref=v1.0.0"
	}
}

func GetDefaultServiceName(smcp version.Version, gatewayName string, className string) string {
	switch smcp {
	case version.SMCP_2_3:
		return gatewayName
	case version.SMCP_2_4:
		return gatewayName
	case version.SMCP_2_5:
		return gatewayName + "-" + className
	case version.SMCP_2_6:
		return gatewayName + "-" + className
	default:
		return gatewayName + "-" + className
	}
}

func GetWaitingCondition(smcp version.Version) string {
	switch smcp {
	case version.SMCP_2_3:
		return "Ready"
	case version.SMCP_2_4:
		return "Ready"
	case version.SMCP_2_5:
		return "Programmed"
	case version.SMCP_2_6:
		return "Programmed"
	default:
		return "Programmed"
	}
}

const GatewayAndRouteYAML = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  annotations:
    networking.istio.io/service-type: "ClusterIP"
  name: gateway
spec:
  gatewayClassName: {{ .GatewayClassName }}
  listeners:
  - name: default
    hostname: "*.example.com"
    port: 8080
    protocol: HTTP
    allowedRoutes:
      namespaces:
        from: All
---
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: http
spec:
  parentRefs:
  - name: gateway
    namespace: foo
  hostnames: ["httpbin.example.com"]
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /get
    backendRefs:
    - name: httpbin
      port: 8000`
