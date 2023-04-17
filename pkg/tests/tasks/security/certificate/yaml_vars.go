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
	"github.com/maistra/maistra-test-tool/pkg/util/env"
)

var (
	sampleCACert  = env.GetRootDir() + "/sampleCerts/ca-cert.pem"
	sampleCAKey   = env.GetRootDir() + "/sampleCerts/ca-key.pem"
	sampleCARoot  = env.GetRootDir() + "/sampleCerts/root-cert.pem"
	sampleCAChain = env.GetRootDir() + "/sampleCerts/cert-chain.pem"
)

var (
	smcpName      string = env.GetDefaultSMCPName()
	meshNamespace string = env.GetDefaultMeshNamespace()
)
