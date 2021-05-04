// generated-from:b1c98792533d4c3379ca57f173bd058cadc51eb73e9b6d8633341c1ae1881aa0 DO NOT REMOVE, DO UPDATE

package client

import "net/http"

// Stubs to keep test errors from happening when bringing in the client.
// This will be overritten the first time you run the openapi-generator
type Configuration struct {
	HTTPClient *http.Client
}

func NewConfiguration() *Configuration {
	return &Configuration{}
}

type APIClient struct {
	cfg *Configuration
}

func NewAPIClient(cfg *Configuration) *APIClient {
	return &APIClient{
		cfg: NewConfiguration(),
	}
}
