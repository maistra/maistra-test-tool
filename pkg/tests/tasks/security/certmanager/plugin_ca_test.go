package certmanager

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/certmanageroperator"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/istio"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestPluginCaCert(t *testing.T) {
	test.NewTest(t).Id("T41").Groups(test.Full, test.ARM).Run(func(t test.TestHelper) {
		//Validate OCP version, this test setup can't be executed in OCP versions less than 4.12
		//More information in: https://57747--docspreview.netlify.app/openshift-enterprise/latest/service_mesh/v2x/ossm-security.html#ossm-cert-manager-integration-istio_ossm-security
		smcpVer := env.GetSMCPVersion()
		if smcpVer.LessThan(version.SMCP_2_4) {
			t.Skip("istio-csr is not supported in SMCP older than v2.4")
		}
		ocpVersion := version.ParseVersion(oc.GetOCPVersion(t))
		if ocpVersion.LessThan(version.OCP_4_12) {
			t.Skip("istio-csr is not supported in OCP older than v4.12")
		}
		if env.GetArch() == "z" || env.GetArch() == "p" {
			t.Skip("istio-csr is not supported for IBM Z&P")
		}

		meshValues := map[string]interface{}{
			"Name":    smcpName,
			"MeshNs":  meshNamespace,
			"Member":  ns.Foo,
			"Version": smcpVer.String(),
			"Rosa":    env.IsRosa(),
		}

		t.Cleanup(func() {
			oc.DeleteFromTemplate(t, meshNamespace, serviceMeshCacertsTmpl, meshValues)
			oc.DeleteFromString(t, meshNamespace, cacerts)
			oc.DeleteSecret(t, meshNamespace, "cacerts")
			oc.RecreateNamespace(t, ns.Foo)
			certmanageroperator.Uninstall(t)
		})

		certmanageroperator.InstallIfNotExist(t)

		t.LogStep("Uninstall existing SMCP")
		oc.RecreateNamespace(t, meshNamespace)

		t.LogStep("Create intermediate CA certificate for Istio")
		oc.ApplyString(t, meshNamespace, cacerts)

		t.LogStep("Deploy SMCP " + smcpVer.String() + " and SMMR")
		oc.ApplyTemplate(t, meshNamespace, serviceMeshCacertsTmpl, meshValues)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Verify that cacerts secret was detected")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.Logs(t, pod.MatchingSelector("app=istiod", meshNamespace), "discovery", assert.OutputContains(
				"Use plugged-in cert at etc/cacerts/tls.key",
				"Istiod detected cacerts secret correctly",
				"Istiod did not detect cacerts secret"))
		})

		t.LogStep("Deploy httpbin and sleep")
		app.InstallAndWaitReady(t, app.Httpbin(ns.Foo), app.Sleep(ns.Foo))

		t.LogStep("Check if httpbin returns 200 OK")
		app.AssertSleepPodRequestSuccess(t, ns.Foo, "http://httpbin:8000/ip")

		t.LogStep("Check mTLS traffic from ingress gateway to httpbin")
		oc.ApplyFile(t, ns.Foo, "https://raw.githubusercontent.com/maistra/istio/maistra-2.6/samples/httpbin/httpbin-gateway.yaml")
		httpbinURL := fmt.Sprintf("http://%s/headers", istio.GetIngressGatewayHost(t, meshNamespace))
		retry.UntilSuccess(t, func(t test.TestHelper) {
			curl.Request(t, httpbinURL, nil, assert.ResponseStatus(http.StatusOK))
		})

		t.LogStep("Check current istiod generation")
		firstGeneration := getIstiodGeneration(t)

		t.LogStep("Trigger CA cert rotation")
		oc.Patch(t, meshNamespace, "certificates", "cacerts", "merge", `{"spec":{"duration":"720h"}}`)

		t.LogStep("Wait until certificates reloaded")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.Logs(t, pod.MatchingSelector("app=istiod", meshNamespace), "discovery", assert.CountExpectedString(
				// expectedOccurrenceNum is 2, because the expected log should appear at startup, so after rotation
				// it should be logged twice.
				"Istiod certificates are reloaded", 2,
				"Istiod detected cacerts secret correctly",
				"Istiod did not detect cacerts secret"))
		})

		// Certificate rotation and logs verification must be repeated to make sure that istiod does not fail after rotation.
		// Checking only the generation is not enough, because istiod might fail a short time after logging that certs are reloaded.
		// This short delay would be caused, because cert watcher triggers reprocessing namespaces and in case of failure,
		// it would not happen immediately.

		t.LogStep("Trigger CA cert rotation")
		oc.Patch(t, meshNamespace, "certificates", "cacerts", "merge", `{"spec":{"duration":"700h"}}`)

		t.LogStep("Wait until certificates reloaded")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.Logs(t, pod.MatchingSelector("app=istiod", meshNamespace), "discovery", assert.CountExpectedString(
				// expectedOccurrenceNum is 2, because the expected log should appear at startup, so after rotation
				// it should be logged twice.
				"Istiod certificates are reloaded", 3,
				"Istiod detected cacerts secret correctly",
				"Istiod did not detect cacerts secret"))
		})

		t.LogStep("Make sure that istiod was not restarted")
		secondGeneration := getIstiodGeneration(t)
		if secondGeneration > firstGeneration {
			t.Errorf("istiod was restarted: old generation: %d, new generation: %d", firstGeneration, secondGeneration)
		}
	})
}

func getIstiodGeneration(t test.TestHelper) int {
	var result int
	retry.UntilSuccess(t, func(t test.TestHelper) {
		generation := shell.Executef(t, "oc get deployments istiod-%s -n %s -o jsonpath='{.metadata.generation}'", smcpName, meshNamespace)
		i, err := strconv.Atoi(generation)
		if err != nil {
			t.Errorf("failed to convert raw generation '%s' to int: %s", generation, err)
		}
		result = i
	})
	return result
}
