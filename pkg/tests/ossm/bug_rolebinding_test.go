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

package ossm

import (
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"

	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

type namespaceResources struct {
	Namespace string
	Resources []string
}

func TestMissingRoleBinding(t *testing.T) {
	NewTest(t).Groups(Full, ARM).Run(func(t TestHelper) {
		t.Log("Verify that role and role binding is detected with SMMR namespace with gateway")
		t.Log("Reference: https://issues.redhat.com/browse/OSSM-2143")

		testResourceTypes := []string{"rolebindings", "roles"}
		namespacesResources := []namespaceResources{
			{Namespace: ns.Bookinfo, Resources: []string{"istio-ingressgateway-sds", "istio-egressgateway-sds"}},
			{Namespace: ns.Foo, Resources: []string{"additional-istio-ingressgateway-sds", "additional-istio-egressgateway-sds"}},
		}

		t.Cleanup(func() {
			oc.Patch(t, meshNamespace, "smcp", smcpName, "json", `[{"op": "remove", "path": "/spec/gateways"}]`)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			app.Uninstall(t, app.Bookinfo(ns.Bookinfo))
		})

		DeployControlPlane(t)
		t.LogStepf("Install %s", ns.Bookinfo)

		app.InstallAndWaitReady(t, app.Bookinfo(ns.Bookinfo))

		checkResources(t, namespacesResources, testResourceTypes, false)

		t.LogStepf("Add gateways to %s namespace and additional gateways to namespace %s", ns.Bookinfo, ns.Foo)
		oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", gatewaysConfig)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		checkResources(t, namespacesResources, testResourceTypes, true)
	})
}

func checkResources(t TestHelper, namespacesResources []namespaceResources, resourceTypes []string, shouldExist bool) {
	for _, nr := range namespacesResources {
		t.LogStepf("Verify that role and rolebinding were created in %s namespace", nr.Namespace)
		for _, resource := range nr.Resources {
			for _, resourceType := range resourceTypes {
				retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(10), func(t TestHelper) {
					if oc.ResourceExists(t, nr.Namespace, resourceType, resource) == shouldExist {
						if shouldExist {
							t.LogSuccessf("%s %s was created in %s namespace", resourceType, resource, nr.Namespace)
						} else {
							t.LogSuccessf("%s %s was not created in %s namespace", resourceType, resource, nr.Namespace)
						}
					} else {
						if shouldExist {
							t.Fatalf("%s %s was not created in %s namespace, but should", resourceType, resource, nr.Namespace)
						} else {
							t.Fatalf("%s %s was created in %s namespace, but should not", resourceType, resource, nr.Namespace)
						}
					}
				})
			}
		}
	}
}

const gatewaysConfig = `
spec:
  gateways:
    egress:
      namespace: bookinfo
    ingress:
      namespace: bookinfo
    additionalIngress:
      additional-istio-ingressgateway:
        enabled: true
        namespace: foo
    additionalEgress:
      additional-istio-egressgateway:
        enabled: true
        namespace: foo
`
