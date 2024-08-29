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

package request

import (
	"net/http"

	"github.com/maistra/maistra-test-tool/pkg/util/curl"
)

func WithHeader(name, value string) curl.RequestOption {
	return headerModifier{
		headers: map[string]string{
			name: value,
		},
	}
}

type headerModifier struct {
	headers map[string]string
}

var _ curl.RequestOption = headerModifier{}

func (m headerModifier) ApplyToRequest(req *http.Request) error {
	for k, v := range m.headers {
		req.Header.Set(k, v)
	}
	return nil
}

func (m headerModifier) ApplyToClient(client *http.Client) error {
	return nil
}
