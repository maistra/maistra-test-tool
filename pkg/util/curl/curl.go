package curl

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type HTTPResponseCheckFunc func(t test.TestHelper, response *http.Response, responseBody []byte, duration time.Duration)

func Request(t test.TestHelper, url string, requestOption RequestOption, checks ...HTTPResponseCheckFunc) []byte {
	t.T().Helper()
	if requestOption == nil {
		requestOption = NilRequestOption{}
	}

	// t.Logf("HTTP request: %s", url)

	startT := time.Now()
	client := &http.Client{}
	if err := requestOption.ApplyToClient(client); err != nil {
		t.Fatalf("failed to modify client: %v", err)
	}

	// Declare HTTP Method and Url
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create HTTP Request: %v", err)
	}
	if err := requestOption.ApplyToRequest(req); err != nil {
		t.Fatalf("failed to modify request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Logf("failed to get HTTP Response: %v", err)
	}

	var responseBody []byte
	if resp != nil {
		defer util.CloseResponseBody(resp)
		responseBody, err = io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
	}

	duration := time.Since(startT)
	// t.Logf("response received in %d ms", duration.Milliseconds())

	for _, check := range checks {
		check(t, resp, responseBody, duration)
	}
	return responseBody
}

type RequestOption interface {
	ApplyToRequest(req *http.Request) error
	ApplyToClient(client *http.Client) error
}

type cookieJarClientModifier struct {
	CookieJar http.CookieJar
}

var _ RequestOption = cookieJarClientModifier{}

func WithCookieJar(jar *cookiejar.Jar) RequestOption {
	return cookieJarClientModifier{CookieJar: jar}
}

func (w cookieJarClientModifier) ApplyToRequest(req *http.Request) error {
	return nil
}

func (w cookieJarClientModifier) ApplyToClient(client *http.Client) error {
	client.Jar = w.CookieJar
	return nil
}

type NilRequestOption struct{}

var _ RequestOption = NilRequestOption{}

func (n NilRequestOption) ApplyToClient(client *http.Client) error {
	return nil
}

func (n NilRequestOption) ApplyToRequest(req *http.Request) error {
	return nil
}
