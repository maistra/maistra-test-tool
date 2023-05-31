package metallb

import (
	_ "embed"
	"fmt"
	"net"
	"strings"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/metallb-operator.yaml
	metallbOperator string

	//go:embed yaml/metallb.yaml
	metallb string
)

func InstallIfNotExist(t test.TestHelper, kubeConfigs ...string) {
	if len(kubeConfigs) == 0 {
		installWithOC(t, *oc.DefaultOC)
		return
	}

	for _, config := range kubeConfigs {
		oc := oc.WithKubeconfig(config)
		if oc == nil {
			t.Errorf("failed to create oc config from kubeconfig file: %s", config)
		}
		installWithOC(t, *oc)
	}
}

func installWithOC(t test.TestHelper, oc oc.OC) {
	installOperator(t, oc)
	deployMetalLB(t, oc)
	createAddressPool(t, oc)
}

func installOperator(t test.TestHelper, oc oc.OC) {
	t.Log("Check if MetalLB controller already exists")
	if oc.ResourceExists(t, ns.MetalLB, "deployments", "metallb-operator-controller-manager") {
		t.Log("MetalLB operator already exists - skip installation of the operator")
		return
	}

	t.Log("Install MetalLB operator")
	oc.ApplyString(t, ns.MetalLB, metallbOperator)
	retry.UntilSuccess(t, func(t test.TestHelper) {
		if !oc.ResourceExists(t, ns.MetalLB, "deployments", "metallb-operator-controller-manager") {
			t.Log("metallb-operator-controller-manager not found - waiting until exists")
		}
	})
	oc.WaitCondition(t, ns.MetalLB, "deployments", "metallb-operator-controller-manager", "Available")
}

func deployMetalLB(t test.TestHelper, oc oc.OC) {
	t.Log("Check if MetalLB controller already exists")
	if oc.ResourceExists(t, ns.MetalLB, "deployments", "controller") {
		t.Log("MetalLB controller already exists - skip deploying MetalLB")
		return
	}

	t.LogStep("Deploy MetalLB")
	oc.ApplyString(t, ns.MetalLB, metallb)
	retry.UntilSuccess(t, func(t test.TestHelper) {
		if !oc.ResourceExists(t, ns.MetalLB, "deployments", "controller") {
			t.Log("MetalLB controller not found - waiting until exists")
		}
	})
	oc.WaitCondition(t, ns.MetalLB, "deployments", "controller", "Available")
}

func createAddressPool(t test.TestHelper, oc oc.OC) {
	t.Log("Check if MetalLB controller already exists")
	if oc.ResourceExists(t, ns.MetalLB, "ipaddresspools", "worker-internal-ips") {
		t.Log("IPAddressPool already exists - skip applying IPAddressPool")
		return
	}

	t.LogStep("Fetch worker internal IPs")
	ipAddrPool := `
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: worker-internal-ips
  namespace: metallb-system
spec:
  addresses:
`
	var ips []string
	retry.UntilSuccess(t, func(t test.TestHelper) {
		out := shell.Execute(t,
			`kubectl get nodes -l node-role.kubernetes.io/worker -o jsonpath='{.items[*].status.addresses[?(@.type=="InternalIP")].address}'`,
			assert.OutputDoesNotContain("Error from server", "Found internal IPs", "failed to get internal node IPs"))
		ips = strings.Fields(out)
		for _, rawIP := range ips {
			if ip := net.ParseIP(rawIP); ip == nil {
				t.Errorf("failed to parse fetched IPs: %s", ips)
			}
			ipAddrPool += fmt.Sprintf("  - %[1]s-%[1]s\n", rawIP)
		}
	})
	t.LogStep("Create IPAddressPool for MetalLB:\n" + ipAddrPool)
	oc.ApplyString(t, ns.MetalLB, ipAddrPool)
}
