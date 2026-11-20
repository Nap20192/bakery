package backend

import (
	"net/http"
	"net/http/httptest"
	"time"
)

const inProcessBaseURL = "http://api.in-process"

func NewInProcess(api http.Handler, timeout time.Duration) *Client {
	return &Client{
		baseURL: inProcessBaseURL,
		http:    &http.Client{Timeout: timeout, Transport: handlerTransport{handler: api}},
	}
}

type handlerTransport struct {
	handler http.Handler
}

func (t handlerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	serverReq := req.Clone(req.Context())
	if serverReq.Body == nil {
		serverReq.Body = http.NoBody
	}
	serverReq.RemoteAddr = "in-process"
	serverReq.RequestURI = req.URL.RequestURI()

	recorder := httptest.NewRecorder()
	t.handler.ServeHTTP(recorder, serverReq)
	resp := recorder.Result()
	resp.Request = req
	return resp, nil
}
