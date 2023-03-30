package istio

import (
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func GetIngressGatewayHost(t test.TestHelper, meshNamespace string) string {
	return shell.Executef(t, "kubectl -n %s get routes istio-ingressgateway -o jsonpath='{.spec.host}'", meshNamespace)
}
