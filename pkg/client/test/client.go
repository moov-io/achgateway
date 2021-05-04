// generated-from:6ea4df8949bf5c2d0502b49ce58cbf2b6669777fab80b7439a6053d4e745664d DO NOT REMOVE, DO UPDATE

package test

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"

	client "github.com/moov-io/ach-conductor/pkg/client"
)

func NewTestClient(handler http.Handler) *client.APIClient {
	mockHandler := MockClientHandler{
		handler: handler,
	}

	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	mockClient := &http.Client{
		Jar: cookieJar,

		// Mock handler that sends the request to the handler passed in and returns the response without a server
		// middleman.
		Transport: &mockHandler,

		// Disables following redirects for testing.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	config := client.NewConfiguration()
	config.HTTPClient = mockClient
	apiClient := client.NewAPIClient(config)

	return apiClient
}

type MockClientHandler struct {
	handler http.Handler
	ctx     *context.Context
}

func (h *MockClientHandler) RoundTrip(request *http.Request) (*http.Response, error) {
	writer := httptest.NewRecorder()

	h.handler.ServeHTTP(writer, request)
	return writer.Result(), nil
}
