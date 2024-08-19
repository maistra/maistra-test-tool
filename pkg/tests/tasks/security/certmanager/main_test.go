// Copyright 2024 Red Hat, Inc.
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

package certmanager

import (
	_ "embed"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	smcpName      = env.GetDefaultSMCPName()
	meshNamespace = env.GetDefaultMeshNamespace()

	//go:embed yaml/istio-csr/istio-ca.yaml
	istioCA string

	//go:embed yaml/istio-csr/mesh.yaml
	serviceMeshIstioCsrTmpl string

	//go:embed yaml/istio-csr/istio-csr.yaml
	istioCsrTmpl string

	//go:embed yaml/cacerts/mesh.yaml
	serviceMeshCacertsTmpl string

	//go:embed yaml/cacerts/cacerts.yaml
	cacerts string
)

func TestMain(m *testing.M) {
	test.NewSuite(m).
		Setup(setupCertManagerOperator).
		Run()
}

func setupCertManagerOperator(t test.TestHelper) {
	ossm.BasicSetup(t)
}
