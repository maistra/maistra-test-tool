package request

import (
	"net/http"

	"github.com/maistra/maistra-test-tool/pkg/util/curl"
)

func WithHost(host string) curl.RequestOption {
	return hostModifier{host: host}
}

type hostModifier struct {
	host string
}

var _ curl.RequestOption = headerModifier{}

func (m hostModifier) ApplyToRequest(req *http.Request) error {
	req.Host = m.host
	return nil
}

func (m hostModifier) ApplyToClient(client *http.Client) error {
	return nil
}
