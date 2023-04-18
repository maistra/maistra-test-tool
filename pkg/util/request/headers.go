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
