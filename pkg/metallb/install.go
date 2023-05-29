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
	t.Log("Check if MetalLB operator already exists")
	output := oc.Get(t, ns.MetalLB, "deployments", "metallb-operator-controller-manager")
	if !strings.Contains(output, "Error from server (NotFound)") {
		t.Log("MetalLB operator already exists - skipping installation")
	} else {
		installOperator(t, oc)
	}
	deployMetalLB(t, oc)
	createAddressPool(t, oc)
}

func installOperator(t test.TestHelper, oc oc.OC) {
	t.Log("Install MetalLB operator")
	oc.ApplyString(t, ns.MetalLB, metallbOperator)
	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.Get(t, ns.MetalLB, "deployments", "metallb-operator-controller-manager", assert.OutputDoesNotContain(
			"Error from server",
			"metallb-operator-controller-manager was found as expected",
			"failed to get metallb-operator-controller-manager"))
	})
	oc.WaitCondition(t, ns.MetalLB, "deployments", "metallb-operator-controller-manager", "Available")
}

func deployMetalLB(t test.TestHelper, oc oc.OC) {
	t.LogStep("Deploy MetalLB")
	oc.ApplyString(t, ns.MetalLB, metallb)
	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.Get(t, ns.MetalLB, "deployments", "controller", assert.OutputDoesNotContain(
			"Error from server",
			"MetalLB controller was found",
			"failed to get MetalLB controller"))
	})
	oc.WaitCondition(t, ns.MetalLB, "deployments", "controller", "Available")
}

func createAddressPool(t test.TestHelper, oc oc.OC) {
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
		// This command fails in zsh, so it may not work on some local environments, therefore it's wrapped in bash -c
		out := shell.Execute(t,
			`bash -c 'kubectl get nodes -l node-role.kubernetes.io/worker -o jsonpath={.items[*].status.addresses[?\(@.type==\"InternalIP\"\)].address}'`,
			assert.OutputDoesNotContain("Error from server", "Found internal IPs", "failed to get internal node IPs"))
		ips = strings.Fields(out)
		for _, rawIP := range ips {
			if ip := net.ParseIP(rawIP); ip == nil {
				t.Errorf("failed to parse fetched IPs: %s", ips)
			}
			ipAddrPool += fmt.Sprintf("  - %[1]s-%[1]s\n", rawIP)
		}
	})
	t.LogStepf("Create IPAddressPool for MetalLB: %s", ipAddrPool)
	oc.ApplyString(t, ns.MetalLB, ipAddrPool)
}
