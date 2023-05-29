package oc

import (
	"fmt"

	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func (o OC) GetServiceClusterIP(t test.TestHelper, ns, serviceName string) string {
	t.T().Helper()
	return o.Invoke(t, fmt.Sprintf("kubectl get service -n %s %s -o jsonpath='{.spec.clusterIP}'", ns, serviceName))
}
