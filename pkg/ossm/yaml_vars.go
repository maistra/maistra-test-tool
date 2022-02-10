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

package ossm

import (
	"github.com/maistra/maistra-test-tool/pkg/util"
)

type SMCP struct {
	Name      string `default:"basic"`
	Namespace string `default:"istio-system"`
}

const (
	jaegerSubYaml = "../templates/olm-templates/nightly/jaeger_subscription.yaml"
	kialiSubYaml = "../templates/olm-templates/nightly/kiali_subscription.yaml"
	ossmSubYaml = "../templates/olm-templates/nightly/ossm_subscription.yaml"
)

var (
	smcpName      string = util.Getenv("SMCPNAME", "basic")
	meshNamespace string = util.Getenv("MESHNAMESPACE", "istio-system")
	smcp          SMCP   = SMCP{smcpName, meshNamespace}
)
