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

package traffic

import (
	"github.com/maistra/maistra-test-tool/pkg/util/env"
)

const (
	// OSSM need custom changes in VirtualService tcp-echo
	echoAllv1Yaml = "../testdata/examples/x86/tcp-echo/tcp-echo-all-v1.yaml"
	echo20v2Yaml  = "../testdata/examples/x86/tcp-echo/tcp-echo-20-v2.yaml"
)

var (
	// OCP4.x
	meshNamespace string = env.Getenv("MESHNAMESPACE", "istio-system")
)
