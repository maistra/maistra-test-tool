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

type RequestOptionList []curl.RequestOption

var _ curl.RequestOption = RequestOptionList{}

func Options(options ...curl.RequestOption) RequestOptionList {
	return options
}

func (l RequestOptionList) ApplyToRequest(req *http.Request) error {
	for _, r := range l {
		if err := r.ApplyToRequest(req); err != nil {
			return err
		}
	}
	return nil
}

func (l RequestOptionList) ApplyToClient(client *http.Client) error {
	for _, r := range l {
		if err := r.ApplyToClient(client); err != nil {
			return err
		}
	}
	return nil
}
