package metallb

import (
	_ "embed"
	"net"
	"strings"

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

	//go:embed yaml/ip-addess-pool.tmpl.yaml
	ipAddressPoolTmpl string
)

func InstallIfNotExist(t test.TestHelper) {
	t.Log("Check if MetalLB already exists")
	output := oc.Get(t, ns.MetalLB, "deployments", "metallb-operator-controller-manager")
	if !strings.Contains(output, "Error from server (NotFound)") {
		t.Log("MetalLB operator already exists - skipping installation")
		return
	}
	installOperator(t)
	deployMetalLB(t)
	createAddressPool(t)
}

func installOperator(t test.TestHelper) {
	t.Log("Install MetalLB operator")
	oc.ApplyString(t, ns.MetalLB, metallbOperator)
	retry.UntilSuccess(t, func(t test.TestHelper) {
		output := oc.Get(t, ns.MetalLB, "deployments", "metallb-operator-controller-manager")
		if strings.Contains(output, "Error from server") {
			t.Errorf("failed to get metallb-operator-controller-manager: %s", output)
		}
	})
	oc.WaitCondition(t, ns.MetalLB, "deployments", "metallb-operator-controller-manager", "Available")
}

func deployMetalLB(t test.TestHelper) {
	t.LogStep("Deploy MetalLB")
	oc.ApplyString(t, ns.MetalLB, metallb)
	retry.UntilSuccess(t, func(t test.TestHelper) {
		output := oc.Get(t, ns.MetalLB, "deployments", "controller")
		if strings.Contains(output, "Error from server") {
			t.Errorf("failed to get metallb controller: %s", output)
		}
	})
	oc.WaitCondition(t, ns.MetalLB, "deployments", "controller", "Available")
}

func createAddressPool(t test.TestHelper) {
	t.LogStep("Fetch worker internal IPs")
	var ips []string
	retry.UntilSuccess(t, func(t test.TestHelper) {
		// This command fails in zsh, so it may not work on some local environments, therefore it's wrapped in bash -c
		out := shell.Execute(t, `bash -c 'kubectl get nodes -l node-role.kubernetes.io/worker -o jsonpath={.items[*].status.addresses[?\(@.type==\"InternalIP\"\)].address}'`)
		if strings.Contains(out, "Error from server") {
			t.Errorf("failed to get worker IPs: %s", out)
		}
		ips = strings.Fields(out)
		for _, rawIP := range ips {
			if ip := net.ParseIP(rawIP); ip == nil {
				t.Errorf("failed to parse fetched IPs: %s", ips)
			}
		}
	})
	t.LogStep("Create IP address pool for MetalLB")
	oc.ApplyTemplate(t, ns.MetalLB, ipAddressPoolTmpl, map[string]interface{}{"IPs": ips})
}
