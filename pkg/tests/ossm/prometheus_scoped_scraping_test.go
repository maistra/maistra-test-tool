package ossm

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/check/require"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestOperatorCanUpdatePrometheusConfigMap(t *testing.T) {
	test.NewTest(t).Groups(test.Full).Run(func(t test.TestHelper) {
		t.Log("This test checks if the operator can update Prometheus ConfigMap when the SMMR is updated")

		t.Cleanup(func() {
			oc.ApplyString(t, meshNamespace, smmr)
		})

		prometheusPodSelector := pod.MatchingSelector("app=prometheus,maistra-control-plane=istio-system", meshNamespace)

		checkPermissionErorr := func(t test.TestHelper) {
			t.LogStep("Check the Prometheus log to see if there is any permission error")
			oc.Logs(t,
				prometheusPodSelector,
				"prometheus",
				assert.OutputDoesNotContain(
					fmt.Sprintf("User \"system:serviceaccount:%s:prometheus\" cannot list resource", meshNamespace),
					"Found no permission error",
					"Expected to find no permission error, but got some error",
				),
			)
		}

		DeployControlPlane(t)

		checkPermissionErorr(t)

		getPrometheusConfigCmd := fmt.Sprintf("oc -n %s get configmap prometheus -o jsonpath='{.data.prometheus\\.yml}'", meshNamespace)

		t.NewSubTest("when creating a SMMR").Run(func(t test.TestHelper) {
			ns := generateNamespace()

			t.Cleanup(func() {
				oc.DeleteNamespace(t, ns)
				restoreDefaultSMMR(t)
			})

			t.LogStepf("Create namespace %s and add it into SMMR", ns)
			oc.CreateNamespace(t, ns)
			updateDefaultSMMRWithNamespace(t, ns)

			t.LogStepf("Look for %s in prometheus ConfigMap", ns)
			retry.UntilSuccess(t, func(t test.TestHelper) {
				shell.Execute(t, getPrometheusConfigCmd, checkForNamespace(ns))
			})
		})

		t.NewSubTest("when adding a new namespace into existing SMMR").Run(func(t test.TestHelper) {
			ns := generateNamespace()
			anotherNs := generateNamespace()

			t.Cleanup(func() {
				oc.DeleteNamespace(t, ns, anotherNs)
				restoreDefaultSMMR(t)
			})

			t.LogStepf("Create namespace %s and add it into SMMR", ns)
			oc.CreateNamespace(t, ns)
			updateDefaultSMMRWithNamespace(t, ns)

			t.LogStepf("Create namespace %s and add it into SMMR along with %s", anotherNs, ns)
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
				restoreDefaultSMMR(t)
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

		t.NewSubTest("when there is no SMMR").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				restoreDefaultSMMR(t)
			})

			t.LogStepf("Delete default SMMR %s", smmr)
			oc.DeleteFromString(t, meshNamespace, smmr)

			checkPermissionErorr(t)
		})

		t.NewSubTest("when the default SMMR with no member").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				restoreDefaultSMMR(t)
			})

			start := time.Now()
			t.LogStepf("Update default SMMR with no member")
			updateDefaultSMMRWithNamespace(t)

			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.LogsSince(t,
					start,
					prometheusPodSelector, "config-reloader",
					assert.OutputContains("Reload triggered",
						"Triggered configuration reloading",
						"Expected to trigger configuration reloading, but did not",
					),
				)
			})

			checkPermissionErorr(t)
		})

		t.NewSubTest("when the default SMMR with nonexistent namespace").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				restoreDefaultSMMR(t)
			})

			t.LogStepf("Update default SMMR with nonexistent member")

			ns := generateNamespace()

			s := buildSMMR(ns)

			t.LogStepf("Update SMMR %s", s)
			oc.ApplyString(t, meshNamespace, s)

			checkPermissionErorr(t)
		})

		t.NewSubTest("query istio_request_total").Run(func(t test.TestHelper) {
			ns := "bookinfo"
			t.Cleanup(func() {
				oc.RecreateNamespace(t, ns)
			})

			t.LogStep("Install bookinfo")
			app.InstallAndWaitReady(t, app.Bookinfo(ns))

			count := 10
			t.LogStepf("Generate %d requests to product page", count)
			productPageURL := app.BookinfoProductPageURL(t, meshNamespace)
			for i := 0; i < count; i++ {
				curl.Request(t, productPageURL, nil)
			}

			t.LogStep(`Check if the "istio_request_total metric is in Prometheus"`)
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.Exec(t,
					prometheusPodSelector,
					"prometheus-proxy",
					"curl localhost:9090/api/v1/query --data-urlencode 'query=istio_requests_total'",
					assert.OutputContains(
						`"__name__":"istio_requests_total"`,
						`Found the "istio_request_total" metric`,
						`Expected to find the "istio_request_total" metric, but found none`,
					),
				)
			})
		})

		t.NewSubTest("[TODO] test under cluster scoped").Run(func(t test.TestHelper) {
			t.Skip()
		})
	})
}

func restoreDefaultSMMR(t test.TestHelper) {
	oc.ApplyString(t, meshNamespace, smmr)
	oc.WaitSMMRReady(t, meshNamespace)
}

func updateDefaultSMMRWithNamespace(t test.TestHelper, names ...string) {
	s := buildSMMR(names...)

	t.LogStepf("Update SMMR %s", s)
	oc.ApplyString(t, meshNamespace, s)
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
		fmt.Sprintf("Found %s in Prometheus config", ns),
		fmt.Sprintf("Expected to find %s in Prometheus config, but not found", ns),
	)
}

func generateNamespace() string {
	return fmt.Sprintf("namespace-%d", rand.Int())
}
