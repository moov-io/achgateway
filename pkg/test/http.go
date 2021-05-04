// generated-from:0fb19a540a93e15a07bd0e3f1f053a620dc7d42baeef277f131e9c64d13d13d7 DO NOT REMOVE, DO UPDATE

package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

func (s TestEnvironment) MakeRequest(method string, target string, body interface{}) *http.Request {
	jsonBody := bytes.Buffer{}
	if body != nil {
		json.NewEncoder(&jsonBody).Encode(body)
	}

	return httptest.NewRequest(method, target, &jsonBody)
}

func (s TestEnvironment) MakeCall(req *http.Request, body interface{}) *http.Response {
	rec := httptest.NewRecorder()
	s.PublicRouter.ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()

	if body != nil {
		json.NewDecoder(res.Body).Decode(&body)
	}

	return res
}
