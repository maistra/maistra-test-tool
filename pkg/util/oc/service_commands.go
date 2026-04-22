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

package oc

import (
	"fmt"

	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func (o OC) GetServiceClusterIP(t test.TestHelper, ns, serviceName string) string {
	t.T().Helper()
	return o.Invoke(t, fmt.Sprintf("kubectl get service -n %s %s -o jsonpath='{.spec.clusterIP}'", ns, serviceName))
}

// GetLoadBalancerAddress returns the external address of a LoadBalancer service.
// Depending on the cloud provider and load balancer type, this may be an IP address or a hostname.
func (o OC) GetLoadBalancerAddress(t test.TestHelper, ns, serviceName string) string {
	t.T().Helper()
	// Try IP first
	addr := o.Invokef(t, `oc -n %s get svc %s -o jsonpath="{.status.loadBalancer.ingress[0].ip}"`, ns, serviceName)
	if addr != "" {
		return addr
	}
	// Try hostname
	return o.Invokef(t, `oc -n %s get svc %s -o jsonpath="{.status.loadBalancer.ingress[0].hostname}"`, ns, serviceName)
}
