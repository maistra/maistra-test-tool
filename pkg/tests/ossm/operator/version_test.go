package operator

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

const EXPECTED_VERSION = "v2.5" // Replace with the expected version

func TestVersion(t *testing.T) {
	test.NewTest(t).Run(func(t test.TestHelper) {
		t.Log("Test to verify helm chart version matches expected version")

		operatorPod := pod.MatchingSelector("name=istio-operator", env.GetOperatorNamespace())
		if operatorPod == nil {
			t.Fatalf("Failed to find istio-operator pod in namespace %s", env.GetOperatorNamespace())
		}

		cmd := exec.Command("kubectl", "exec", "deploy/istio-operator", "-n", env.GetOperatorNamespace(),
			"--", "cat", "/usr/local/share/istio-operator/helm/", env.GetSMCPVersion().String(),
			"/istio-control/istio-discovery/templates/deployment.yaml", "|", "grep", "maistra-version", "|", "awk", "'{print $2}'")

		outputBytes, err := cmd.Output()
		if err != nil {
			t.Fatalf("Failed to execute command: %v", err)
		}

		output := strings.TrimSpace(string(outputBytes))

		if output != EXPECTED_VERSION {
			t.Fatalf("Version mismatch: expected %s, got %s", EXPECTED_VERSION, output)
		}

		t.Logf("Version matches expected version: %s", EXPECTED_VERSION)
	})
}
