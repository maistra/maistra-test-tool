package ossm

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/check/require"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestOperatorCanUpdatePrometheusConfigMap(t *testing.T) {
	test.NewTest(t).Groups(test.Full).Run(func(t test.TestHelper) {
		hack.DisableLogrusForThisTest(t)

		t.Log("This test checks if the operator can update Prometheus ConfigMap when the SMMR is updated")

		meshNamespace := env.GetDefaultMeshNamespace()
		defaultSMMR := shell.Executef(t, "oc -n %s get smmr default -o jsonpath='{.metadata.annotations.kubectl\\.kubernetes\\.io/last-applied-configuration}'", meshNamespace)

		t.Cleanup(func() {
			oc.ApplyString(t, meshNamespace, defaultSMMR)
		})

		t.LogStepf("Delete current SMMR %s", defaultSMMR)
		oc.DeleteFromString(t, meshNamespace, defaultSMMR)

		getPrometheusConfigCmd := fmt.Sprintf("oc -n %s get configmap prometheus -o jsonpath='{.data.prometheus\\.yml}'", meshNamespace)

		t.NewSubTest("when creating a SMMR").Run(func(t test.TestHelper) {
			ns := generateNamespace()

			t.Cleanup(func() {
				oc.DeleteNamespace(t, ns)
			})

			t.LogStepf("Create namespace %s and add it into SMMR", ns)
			oc.CreateNamespace(t, ns)
			updateDefaultSMMRWithNamespace(t, ns)

			t.LogStepf("Look for %s in prometheus ConfigMap", ns)
			shell.Execute(t, getPrometheusConfigCmd, checkForNamespace(ns))
		})

		t.NewSubTest("when adding a new namespace into existing SMMR").Run(func(t test.TestHelper) {
			ns := generateNamespace()
			anotherNs := generateNamespace()

			t.Cleanup(func() {
				oc.DeleteNamespace(t, ns, anotherNs)
			})

			t.LogStepf("Create namespace %s and add it into SMMR", ns)
			oc.CreateNamespace(t, ns)
			updateDefaultSMMRWithNamespace(t, ns)

			oc.CreateNamespace(t, anotherNs)
			updateDefaultSMMRWithNamespace(t, ns, anotherNs)

			t.LogStepf("Look for %s in prometheus ConfigMap", []string{ns, anotherNs})
			retry.UntilSuccess(t, func(t test.TestHelper) {
				shell.Execute(t, getPrometheusConfigCmd, checkForNamespace(ns), checkForNamespace(anotherNs))
			})
		})

		t.NewSubTest("when removing a namespace from existing SMMR").Run(func(t test.TestHelper) {
			ns := generateNamespace()
			anotherNs := generateNamespace()

			t.Cleanup(func() {
				oc.DeleteNamespace(t, ns, anotherNs)
			})

			t.LogStepf("Create namespace (%s,%s) and add it into SMMR", ns, anotherNs)
			oc.CreateNamespace(t, ns, anotherNs)
			updateDefaultSMMRWithNamespace(t, ns, anotherNs)

			t.LogStepf("Update SMMR with only %s", ns)
			updateDefaultSMMRWithNamespace(t, ns)

			t.LogStepf("Look for %s in prometheus ConfigMap", ns)
			retry.UntilSuccess(t, func(t test.TestHelper) {
				shell.Execute(t, getPrometheusConfigCmd,
					checkForNamespace(ns),
					require.OutputDoesNotContain(anotherNs,
						fmt.Sprintf("Expected to not find %s in the Prometheus config", anotherNs),
						fmt.Sprintf("Found unexpected %s in the Prometheus config", anotherNs),
					),
				)
			})
		})

		t.NewSubTest("[TODO] test under cluster scoped").Run(func(t test.TestHelper) {})
	})
}

func updateDefaultSMMRWithNamespace(t test.TestHelper, names ...string) {
	oc.ApplyString(t, meshNamespace, buildSMMR(names...))
	oc.WaitSMMRReady(t, meshNamespace)
}

func buildSMMR(names ...string) string {
	yaml := `
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:`

	for _, name := range names {
		yaml += fmt.Sprintf(`
  - %s`, name)
	}

	return yaml
}

func checkForNamespace(ns string) common.CheckFunc {
	return require.OutputContains(ns,
		fmt.Sprintf("found %s in Prometheus config", ns),
		fmt.Sprintf("expected to find %s in Prometheus config, but not found", ns),
	)
}

func generateNamespace() string {
	return fmt.Sprintf("namespace-%d", rand.Int())
}
