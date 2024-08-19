// Copyright 2024 Red Hat, Inc.
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

	//go:embed yaml/kiali-cluster-monitoring-view.tmpl.yaml
	kialiClusterMonitoringView string

	//go:embed yaml/network-policy.yaml
	networkPolicy string
)

func TestMain(m *testing.M) {
	test.NewSuite(m).
		Setup(ossm.BasicSetup).
		Run()
}
