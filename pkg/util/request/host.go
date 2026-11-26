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

// WithHost overrides the Host header of the request while connecting to the
// host in the URL. The URL host must be directly resolvable and reachable
// from where the test runs, because requests with an overridden Host header
// cannot be sent through an HTTP proxy: net/http builds the absolute-form
// request-target for proxied requests from req.Host (request.go: "host :=
// r.Host; if host == \"\" { host = r.URL.Host }"), so the override would leak
// into the request-target and the proxy would try to resolve the (typically
// fake) overridden host instead of the URL host. Bypassing the proxy keeps
// the override confined to the Host header, which is all it is meant to be.
func WithHost(host string) curl.RequestOption {
	return hostModifier{host: host}
}

type hostModifier struct {
	host string
}

var _ curl.RequestOption = hostModifier{}

func (m hostModifier) ApplyToRequest(req *http.Request) error {
	req.Host = m.host
	return nil
}

func (m hostModifier) ApplyToClient(client *http.Client) error {
	// Only install the proxyless transport when none is set yet, so that a
	// transport installed by another option composed with this one (e.g.
	// WithTLS, which is proxyless already) is not clobbered. The zero-value
	// Transport has no Proxy, so ProxyFromEnvironment never applies.
	if client.Transport == nil {
		client.Transport = &http.Transport{}
	}
	return nil
}
