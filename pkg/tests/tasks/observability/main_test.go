package observability

import (
	_ "embed"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	smcpName                 = env.GetDefaultSMCPName()
	meshNamespace            = env.GetDefaultMeshNamespace()
	monitoringNs             = "openshift-monitoring"
	userWorkloadMonitoringNs = "openshift-user-workload-monitoring"

	//go:embed yaml/cluster-monitoring-config.tmpl.yaml
	clusterMonitoringConfigTmpl string

	//go:embed yaml/enable-prometheus-metrics.yaml
	enableTrafficMetrics string

	//go:embed yaml/istiod-monitor.yaml
	istiodMonitor string

	//go:embed yaml/istio-proxy-monitor.yaml
	istioProxyMonitor string

	//go:embed yaml/federated-monitor.yaml
	federatedMonitor string

	//go:embed yaml/mesh.tmpl.yaml
	meshTmpl string

	//go:embed yaml/federated-monitoring-mesh.tmpl.yaml
	federatedMonitoringMeshTmpl string

	//go:embed yaml/kiali-user-workload-monitoring.tmpl.yaml
	kialiUserWorkloadMonitoringTmpl string

	//go:embed yaml/network-policy.yaml
	networkPolicy string
)

func TestMain(m *testing.M) {
	test.NewSuite(m).
		Setup(ossm.BasicSetup).
		Run()
}
