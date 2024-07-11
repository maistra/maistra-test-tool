// Copyright 2021 Red Hat, Inc.
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
	_ "embed"
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tempo"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/template"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestTempoTracing(t *testing.T) {
	test.NewTest(t).Groups(test.Full, test.ARM).Run(func(t test.TestHelper) {
		if env.GetSMCPVersion().LessThan(version.SMCP_2_5) {
			t.Skip("Skipped because integration with tempo is available only in v2.5+")
		}
		if env.GetArch() == "z" {
			t.Skip("Tempo tracing is not supported in IBM Z platform")
		}

		meshValues := map[string]interface{}{
			"Name":          smcpName,
			"Version":       env.GetSMCPVersion().String(),
			"Rosa":          env.IsRosa(),
			"OtelNamespace": ns.Bookinfo,
		}

		t.Cleanup(func() {
			tempo.Uninstall(t)
			oc.RecreateNamespace(t, ns.Bookinfo)
			oc.RecreateNamespace(t, meshNamespace)
		})

		t.LogStep("Install TempoStack")
		tempo.InstallIfNotExist(t)
		t.LogStep("TempoStack was installed")
		oc.RecreateNamespace(t, meshNamespace)

		t.LogStep("Install SMCP with otel extensionProviders")
		oc.ReplaceOrApplyString(t, meshNamespace, template.Run(t, externalTracingSMCP, meshValues))
		oc.WaitSMCPReady(t, meshNamespace, smcpName)
		oc.ApplyString(t, meshNamespace, GetSMMRTemplate())
		oc.WaitSMMRReady(t, meshNamespace)

		t.LogStep("Intalling Bookinfo app")
		app.InstallAndWaitReady(t, app.Bookinfo(ns.Bookinfo))

		t.LogStep("Create open telemetry collector in bookinfo namespace")
		oc.ReplaceOrApplyString(t, ns.Bookinfo, template.Run(t, otel, map[string]interface{}{"TracingNamespace": tempo.GetTracingNamespace()}))
		retry.UntilSuccess(t, func(t test.TestHelper) {
			t.T().Helper()
			oc.WaitPodReady(t, pod.MatchingSelector("app.kubernetes.io/component=opentelemetry-collector", ns.Bookinfo))
		})

		t.LogStep("Create telemetry cr in SMCP namespace")
		oc.ReplaceOrApplyString(t, meshNamespace, template.Run(t, telemetry, nil))

		t.LogStep("Generate request to product page")
		curl.Request(t, app.BookinfoProductPageURL(t, meshNamespace), nil)

		t.LogStepf("Check that Tempostack contain traces")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			checkThatTracesForServiceExist(t, "productpage."+ns.Bookinfo)
		})
		checkThatTracesForServiceExist(t, "details."+ns.Bookinfo)
		checkThatTracesForServiceExist(t, "reviews."+ns.Bookinfo)
		checkThatTracesForServiceExist(t, "istio-ingressgateway."+meshNamespace)
	})
}

func checkThatTracesForServiceExist(t test.TestHelper, service string) {
	// check traces directly in the query frontend pod
	oc.Exec(t,
		pod.MatchingSelector("app.kubernetes.io/component=query-frontend", tempo.GetTracingNamespace()),
		"tempo-query",
		fmt.Sprintf("curl -sS \"localhost:16686/api/traces?limit=5&lookback=10m&service=%s\"", service),
		assert.OutputDoesNotContain(
			"\"data\":[]",
			fmt.Sprintf("Response contains traces for the service %s", service),
			fmt.Sprintf("Response doesn't contain any traces for the service %s", service),
		),
		assert.OutputContains(
			fmt.Sprintf("\"serviceName\":\"%s\"", service),
			"Response contains traces with expected message",
			"Response doesn't contain trace with expected mesage",
		),
		assert.OutputContains(
			"outbound|9080||",
			"Response contains traces with expected message",
			"Response doesn't contain trace with expected mesage",
		))
}

const (
	externalTracingSMCP = `
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
spec:
  version: {{ .Version }}
  general:
    validationMessages: true 
  tracing:
    type: None
  addons:
    grafana:
      enabled: false
    kiali:
      enabled: false
    prometheus:
      enabled: false
  meshConfig:
    extensionProviders:
      - name: otel
        opentelemetry:
          port: 4317
          service: "otel-collector.{{ .OtelNamespace }}.svc.cluster.local"
  {{ if .Rosa }} 
  security:
    identity:
      type: ThirdParty
  {{ end }}`
)

const (
	otel = `
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: otel
spec:
  mode: deployment
  config: |
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
    exporters:
      otlp:
        endpoint: "tempo-sample-distributor.{{ .TracingNamespace }}.svc.cluster.local:4317"
        tls:
          insecure: true
    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: []
          exporters: [otlp]`
)

const (
	telemetry = `
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
metadata:
  name: mesh-default
spec:
  tracing:
  - providers:
    - name: otel
    randomSamplingPercentage: 100`
)
