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

package migration

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

const maistraIgnoreLabel = `maistra.io/ignore-namespace="true"`

var (
	meshNamespace    = env.GetDefaultMeshNamespace()
	migrationGateway = "yaml/bookinfo-gateway.yaml"
)

var (
	//go:embed yaml/enable-strict-mtls-peer-auth.yaml
	enableMTLSPeerAuth string

	//go:embed yaml/istio-ca.yaml
	istioCA string

	//go:embed yaml/mesh.yaml
	serviceMeshIstioCSRTmpl string

	//go:embed yaml/istio-csr.yaml
	istioCSRTmpl string

	//go:embed yaml/istio-cert-manager.yaml
	istioWithCertManager string

	//go:embed yaml/mesh-custom-ca.yaml
	serviceMeshCustomCATmpl string

	//go:embed yaml/istio-custom-ca.yaml
	istioCustomCATmpl string
)

func TestMain(m *testing.M) {
	test.NewSuite(m).
		Setup(ossm.BasicSetup).
		Run()
}

// TODO: These helper methods/structs can potentially move to other packages.

type ingressStatus struct {
	IP       string `json:"ip,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

func (i ingressStatus) GetHostname() string {
	if i.IP != "" {
		return i.IP
	} else if i.Hostname != "" {
		return i.Hostname
	}
	return ""
}

func workloadNames(workloads []workload) []string {
	var names []string
	for _, wk := range workloads {
		names = append(names, wk.Name)
	}
	return names
}

type workload struct {
	Name   string
	Labels map[string]string
}

func toSelector(labels map[string]string) string {
	var parts []string
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}

// Will continually request the URL until the test has ended and assert for success.
// Once the test is over, this func will clean itself up and wait until in flight
// requests have finished.
func continuallyRequest(t test.TestHelper, url string) {
	t.T().Helper()
	t.Logf("Continually requesting URL: %s", url)

	ctx, cancel := context.WithCancel(context.Background())
	// 1. cancel
	// 2. wait for in flight requests to finish so that the curl assertion doesn't fail the test if underlying resources have been deleted.
	// 3. continue with other cleanups like deletion of resources.
	stopped := make(chan struct{})
	t.Cleanup(func() {
		cancel()
		t.Log("Waiting for continual requests to stop...")
		<-stopped
		t.Log("Continual requests stopped.")
	})
	go func(ctx context.Context) {
	ReqLoop:
		for {
			if t.Failed() {
				t.Log("Ending continual requests. Test failed.")
				break
			}

			select {
			case <-ctx.Done():
				t.Log("Ending continual requests. Context has been cancelled.")
				break ReqLoop
			case <-time.After(time.Millisecond * 500):
				curl.Request(t, url, curl.WithContext(ctx), assert.RequestSucceedsAndIgnoreContextCancelled("productpage request failed"))
			}
		}
		stopped <- struct{}{}
	}(ctx)
}

func setupIstio(t test.TestHelper, istios ...ossm.Istio) {
	t.T().Helper()
	t.Cleanup(func() {
		t.Log("Cleaning up IstioCNI")
		oc.DeleteResource(t, "", "IstioCNI", "default")
	})
	for _, istio := range istios {
		istio := istio
		t.Cleanup(func() {
			t.Logf("Cleaning up Istio %s", istio.Name)
			oc.DeleteResource(t, "", "Istio", istio.Name)
		})
		if istio.Template == "" {
			istio.Template = `apiVersion: sailoperator.io/v1
kind: Istio
metadata:
  name: {{ .Name }}
spec:
  namespace: {{ .Namespace }}`
		}
		oc.ApplyTemplate(t, "", istio.Template, istio)
		oc.Label(t, "", "Istio", istio.Name, oc.MaistraTestLabel+`=""`)
		oc.DefaultOC.WaitFor(t, "", "Istio", istio.Name, "condition=Ready")
	}
	oc.CreateNamespace(t, "istio-cni")
	istioCNI := `apiVersion: sailoperator.io/v1
kind: IstioCNI
metadata:
  name: default
spec:
  namespace: istio-cni`
	oc.ApplyString(t, "", istioCNI)
	oc.DefaultOC.WaitFor(t, "", "IstioCNI", "default", "condition=Ready")
}

// Returns either the ip address or the hostname of the LoadBalancer from the Service status.
// Fails if neither exist.
func getLoadBalancerServiceHostname(t test.TestHelper, name string, namespace string) ingressStatus {
	t.T().Helper()
	resp := oc.GetJson(t, ns.Bookinfo, "Service", "bookinfo-gateway", "{.status.loadBalancer.ingress}")
	var v []ingressStatus
	if err := json.Unmarshal([]byte(resp), &v); err != nil {
		t.Fatalf("Unable to unmarshal ingress status from Service response: %s", err)
	}
	if got := len(v); got != 1 {
		t.Fatalf("Expected there to be a 1 ingress but there are: %d", got)
	}
	status := v[0]

	if status.GetHostname() == "" {
		t.Fatalf("Service: %s/%s has neither an ip or hostname", name, namespace)
	}

	return status
}

func namespaceInSMMR(t test.TestHelper, namespace string, smmrName string, smmrNamespace string) bool {
	t.T().Helper()
	var members []string
	t.Log("Checking if \"bookinfo\" has been removed from default SMMR...")
	output := oc.GetJson(t, smmrNamespace, "ServiceMeshMemberRoll", smmrName, "{.status.configuredMembers}")
	if err := json.Unmarshal([]byte(output), &members); err != nil {
		t.Error(err)
	}
	for _, member := range members {
		if member == namespace {
			return true
		}
	}
	return false
}

// Ensure stable looks at the resource version of a resource and ensures it isn't constantly changing
// e.g. two controllers fighting each other by continually updating the same resource.
func ensureResourceStable(t test.TestHelper, name string, namespace string, kind string) {
	t.T().Helper()
	// For a period of time, ensure the resource version stays the same.
	// If the resource version changes, reset the clock.
	// Go until either it is stable or the max time limit has been reached.
	stablePeriod := time.Second * 5
	maxTimeout := time.NewTimer(time.Second * 30)

	for {
		currentRV := oc.GetJson(t, namespace, kind, name, "{.metadata.resourceVersion}")
		select {
		case <-maxTimeout.C:
			t.Fatalf("Max time limit reached. Resource: %s %s/%s is not stable", kind, namespace, name)
			return
		case <-time.After(stablePeriod):
			newRV := oc.GetJson(t, namespace, kind, name, "{.metadata.resourceVersion}")
			if currentRV == newRV {
				t.Logf("Resource: %s %s/%s is stable", kind, namespace, name)
				return
			}
		}
	}
}
