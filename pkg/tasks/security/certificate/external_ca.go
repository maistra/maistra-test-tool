// Copyright 2021 Red Hat, Inc.
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

package certificate

import (
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupExternalCert() {
	util.Log.Info("Cleanup")
	sleep := examples.Sleep{"foo"}
	httpbin := examples.Httpbin{"foo"}
	sleep.Uninstall()
	httpbin.Uninstall()
	util.Shell(`kubectl -n istio-system delete secret cacerts`)
	util.Shell(`kubectl -n istio-system patch smcp/basic --type=json -p='[{"op": "remove", "path": "/spec/security/certificateAuthority"}]'`)
	util.Shell(`oc -n istio-system wait --for condition=Ready smcp/basic --timeout 180s`)
	util.Shell(`kubectl patch -n istio-system smcp/basic --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":false},"controlPlane":{"mtls":false}}}}'`)
	util.Shell(`oc -n istio-system wait --for condition=Ready smcp/basic --timeout 180s`)
	time.Sleep(time.Duration(20) * time.Second)
}

func TestExternalCert(t *testing.T) {
	defer cleanupExternalCert()

	util.Log.Info("Test External Certificates")
	util.Log.Info("Enable Control Plane MTLS")
	util.Shell(`kubectl patch -n istio-system smcp/basic --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":true},"controlPlane":{"mtls":true}}}}'`)
	util.Shell(`oc -n istio-system wait --for condition=Ready smcp/basic --timeout 180s`)

	httpbin := examples.Httpbin{"foo"}
	httpbin.Install()

	t.Run("Security_plugging_external_cert_test", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Adding an external CA")
		util.Shell(`kubectl create -n %s secret generic cacerts --from-file=%s --from-file=%s --from-file=%s --from-file=%s`,
			"istio-system", sampleCACert, sampleCAKey, sampleCARoot, sampleCAChain)

		util.Shell(`kubectl -n istio-system patch smcp/basic --type=merge --patch="%s"`, CertSMCPPath)
		util.Shell(`oc -n istio-system wait --for condition=Ready smcp/basic --timeout 180s`)
		time.Sleep(time.Duration(20) * time.Second)

		sleep := examples.Sleep{"foo"}
		sleep.Install()
		sleepPod, err := util.GetPodName("foo", "app=sleep")
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		time.Sleep(time.Duration(20) * time.Second)

		util.Log.Info("Verify the new certificates")
		util.ShellMuteOutput(`kubectl exec -n %s -it %s -c istio-proxy -- /bin/cat /var/run/secrets/istio/root-cert.pem > /tmp/pod-root-cert.pem`, "foo", sleepPod)
		//util.ShellMuteOutput(`kubectl exec -n %s -it %s -c istio-proxy -- /bin/cat /etc/certs/cert-chain.pem > /tmp/pod-cert-chain.pem`, "foo", sleepPod)

		util.Log.Info("Verify the root certificate")
		util.ShellMuteOutput(`openssl x509 -in %s -text -noout > /tmp/root-cert.crt.txt`, sampleCARoot)
		util.ShellMuteOutput(`openssl x509 -in /tmp/pod-root-cert.pem -text -noout > /tmp/pod-root-cert.crt.txt`)
		if err := util.CompareFiles("/tmp/root-cert.crt.txt", "/tmp/pod-root-cert.crt.txt"); err != nil {
			t.Error("Error root cert.")
		}
	})
}
