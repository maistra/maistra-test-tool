package request

import (
	"net/http"

	"github.com/maistra/maistra-test-tool/pkg/util/curl"
)

type RequestOptionList []curl.RequestOption

var _ curl.RequestOption = RequestOptionList{}

func Options(options ...curl.RequestOption) RequestOptionList {
	return RequestOptionList(options)
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
