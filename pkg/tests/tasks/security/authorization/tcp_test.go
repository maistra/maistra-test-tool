// Copyright 2023 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authorization

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

// TestAuthorizationTCPTraffic validates authorization polices for TCP traffic.
func TestAuthorizationTCPTraffic(t *testing.T) {
	test.NewTest(t).Id("T21").Groups(test.Full, test.InterOp).Run(func(t test.TestHelper) {
		hack.DisableLogrusForThisTest(t)

		ns := "foo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		t.Log("This test validates authorization policies for TCP traffic.")
		t.Log("Doc reference: https://istio.io/v1.14/docs/tasks/security/authorization/authz-tcp/")

		t.LogStep("Install sleep and echo")
		app.InstallAndWaitReady(t, app.Sleep(ns), app.Echo(ns))

		t.LogStep("Verify sleep to echo TCP connections")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			assertPortTcpEchoAccepted(t, ns, "9000")
			assertPortTcpEchoAccepted(t, ns, "9001")
		})

		retry.UntilSuccess(t, func(t test.TestHelper) {
			assertPortTcpEchoIPAccepted(t, ns, "9002")
		})

		t.NewSubTest("TCP ALLOW policy").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, TCPAllowPolicy)
			})
			t.LogStep("Apply a policy to allow tcp requests to port 9000 and 9001")
			oc.ApplyString(t, ns, TCPAllowPolicy)

			retry.UntilSuccess(t, func(t test.TestHelper) {
				assertPortTcpEchoAccepted(t, ns, "9000")
				assertPortTcpEchoAccepted(t, ns, "9001")
			})

			retry.UntilSuccess(t, func(t test.TestHelper) {
				assertPortTcpEchoIPDenied(t, ns, "9002")
			})
		})

		t.NewSubTest("TCP invalid policy").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, TCPAllowGETPolicy)
			})
			t.LogStep("Apply an invalid policy to allow requests to port 9000 and add an HTTP GET field")
			oc.ApplyString(t, ns, TCPAllowGETPolicy)

			retry.UntilSuccess(t, func(t test.TestHelper) {
				assertPortTcpEchoDenied(t, ns, "9000")
				assertPortTcpEchoDenied(t, ns, "9001")
			})
		})

		t.NewSubTest("TCP deny policy").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, TCPDenyGETPolicy)
			})
			t.LogStep("Apply a policy to deny tcp requests to port 9000")
			oc.ApplyString(t, ns, TCPDenyGETPolicy)

			retry.UntilSuccess(t, func(t test.TestHelper) {
				assertPortTcpEchoDenied(t, ns, "9000")
				assertPortTcpEchoAccepted(t, ns, "9001")
			})
		})
	})
}

func assertPortTcpEchoAccepted(t test.TestHelper, ns string, port string) {
	cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc tcp-echo %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`,
		port, port)
	oc.Exec(t,
		pod.MatchingSelector("app=sleep", ns),
		"sleep",
		cmd,
		assert.OutputContains(
			"connection succeeded",
			fmt.Sprintf("Got expected hello message on port %s", port),
			fmt.Sprintf("Expected return message hello, but failed on port %s", port)))
}

func assertPortTcpEchoDenied(t test.TestHelper, ns string, port string) {
	cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc tcp-echo %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`,
		port, port)
	oc.Exec(t,
		pod.MatchingSelector("app=sleep", ns),
		"sleep",
		cmd,
		assert.OutputContains(
			"connection rejected",
			fmt.Sprintf("Got expected connection rejected on port %s", port),
			fmt.Sprintf("Expected connection rejected, but got return message hello on port %s", port)))
}

func assertPortTcpEchoIPAccepted(t test.TestHelper, ns string, port string) {
	podIP := oc.GetPodIP(t,
		pod.MatchingSelector("app=tcp-echo", ns))
	cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc %s %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`,
		port, podIP, port)
	oc.Exec(t,
		pod.MatchingSelector("app=sleep", ns),
		"sleep",
		cmd,
		assert.OutputContains(
			"connection succeeded",
			fmt.Sprintf("Got expected hello message on port %s", port),
			fmt.Sprintf("Expected return message hello, but failed on port %s", port)))
}

func assertPortTcpEchoIPDenied(t test.TestHelper, ns string, port string) {
	podIP := oc.GetPodIP(t,
		pod.MatchingSelector("app=tcp-echo", ns))
	cmd := fmt.Sprintf(`sh -c 'echo "port %s" | nc %s %s' | grep "hello" && echo 'connection succeeded' || echo 'connection rejected'`,
		port, podIP, port)
	oc.Exec(t,
		pod.MatchingSelector("app=sleep", ns),
		"sleep",
		cmd,
		assert.OutputContains(
			"connection rejected",
			fmt.Sprintf("Got expected connection rejected on port %s", port),
			fmt.Sprintf("Expected connection rejected, but got return message hello on port %s", port)))
}

const (
	TCPAllowPolicy = `
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: tcp-policy
  namespace: foo
spec:
  selector:
    matchLabels:
      app: tcp-echo
  action: ALLOW
  rules:
  - to:
    - operation:
       ports: ["9000", "9001"]
`

	TCPAllowGETPolicy = `
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: tcp-policy
  namespace: foo
spec:
  selector:
    matchLabels:
      app: tcp-echo
  action: ALLOW
  rules:
  - to:
    - operation:
        methods: ["GET"]
        ports: ["9000"]
`

	TCPDenyGETPolicy = `
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: tcp-policy
  namespace: foo
spec:
  selector:
    matchLabels:
      app: tcp-echo
  action: DENY
  rules:
  - to:
    - operation:
        methods: ["GET"]
        ports: ["9000"]
`
)
