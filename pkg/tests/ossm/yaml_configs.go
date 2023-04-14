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

import _ "embed"

var (
	//go:embed yaml/smcp_minimal.yaml
	smcp_minimal_template string

	//go:embed yaml/smcp_full.yaml
	smcp_full_template string

	//go:embed yaml/smmr.yaml
	smmr string
)

func GetSMCPTemplates() map[string]string {
	return map[string]string{
		"minimal": smcp_minimal_template,
		"full":    smcp_full_template,
	}
}

func GetSMMRTemplate() string {
	return smmr
}
