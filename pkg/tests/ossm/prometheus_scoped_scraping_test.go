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
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	namespaces "github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

var prometheusPodSelector oc.PodLocatorFunc = pod.MatchingSelector("app=prometheus,maistra-control-plane=istio-system", meshNamespace)

func TestOperatorCanUpdatePrometheusConfigMap(t *testing.T) {
	test.NewTest(t).Groups(test.Full).Run(func(t test.TestHelper) {
		t.Log("This test checks if the operator can update Prometheus ConfigMap when the SMMR is updated")

		if env.GetSMCPVersion().LessThan(version.SMCP_2_4) {
			t.Skip("Test only valid in SMCP versions v2.4+")
		}

		t.Cleanup(func() {
			oc.ApplyString(t, meshNamespace, smmr)
		})

		DeployControlPlane(t)

		checkPermissionError(t)

		t.NewSubTest("when the default SMMR with no member").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				restoreDefaultSMMR(t)
			})

			start := time.Now()
			t.LogStepf("Update default SMMR with no member")
			updateDefaultSMMRWithNamespace(t)

			checkConfigurationReloadingTriggered(t, start)
			checkPermissionError(t)
		})

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
			testPrometheusConfigWithAsserts(t, assertConfigMapContainsNamespace(ns))
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
			testPrometheusConfigWithAsserts(t, assertConfigMapContainsNamespace(ns), assertConfigMapContainsNamespace(anotherNs))
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
			testPrometheusConfigWithAsserts(t, assertConfigMapContainsNamespace(ns), assertConfigMapDoesNotContainNamespace(anotherNs))
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

			testPrometheusConfigWithAsserts(t, assertConfigMapDoesNotContainNamespace(ns))
			checkPermissionError(t)
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

		t.NewSubTest("when removing SMMR").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				restoreDefaultSMMR(t)
			})

			start := time.Now()
			t.LogStepf("Delete default SMMR \n%s", smmr)
			oc.DeleteFromString(t, meshNamespace, smmr)

			testPrometheusConfigWithAsserts(t,
				assertConfigMapDoesNotContainNamespace(namespaces.Bar),
				assertConfigMapDoesNotContainNamespace(namespaces.Bookinfo),
				assertConfigMapDoesNotContainNamespace(namespaces.Foo),
				assertConfigMapDoesNotContainNamespace(namespaces.Legacy),
			)
			checkConfigurationReloadingTriggered(t, start)
			checkPermissionError(t)
		})

		t.NewSubTest("[TODO] test under cluster scoped").Run(func(t test.TestHelper) {
			t.Skip()
		})
	})
}

func checkPermissionError(t test.TestHelper) {
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

func checkConfigurationReloadingTriggered(t test.TestHelper, start time.Time) {
	// By default, any changes in the `ConfigMap`, the kubelet will sync them to the mapped volume on one minute interval.
	t.Log("Wait one minute on the kubelet to update the volume to reflect the changes")
	time.Sleep(1 * time.Minute)
	retry.UntilSuccessWithOptions(t, retry.Options().DelayBetweenAttempts(5*time.Second).MaxAttempts(13), func(t test.TestHelper) {
		oc.LogsSince(t,
			start,
			prometheusPodSelector, "config-reloader",
			assert.OutputContains("Reload triggered",
				"Triggered configuration reloading",
				fmt.Sprintf("Expected to trigger configuration reloading, but did not. Start time: %s", start.String()),
			),
		)
	})
}

func testPrometheusConfigWithAsserts(t test.TestHelper, asserts ...common.CheckFunc) {
	retry.UntilSuccess(t, func(t test.TestHelper) {
		shell.Execute(t,
			fmt.Sprintf("oc -n %s get configmap prometheus -o jsonpath='{.data.prometheus\\.yml}'", meshNamespace),
			asserts...)
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

func assertConfigMapContainsNamespace(ns string) common.CheckFunc {
	return require.OutputContains(ns,
		fmt.Sprintf("Found %s in Prometheus config", ns),
		fmt.Sprintf("Expected to find %s in Prometheus config, but not found", ns),
	)
}

func assertConfigMapDoesNotContainNamespace(ns string) common.CheckFunc {
	return require.OutputDoesNotContain(ns,
		fmt.Sprintf("Expected to not find %s in the Prometheus config", ns),
		fmt.Sprintf("Found unexpected %s in the Prometheus config", ns),
	)
}

func generateNamespace() string {
	return fmt.Sprintf("namespace-%d", rand.Int())
}
