package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testToken = "abc123"

func testCheckHeaders(t *testing.T, r *http.Request) {
	if r.Method != http.MethodGet {
		t.Errorf("handler: want: %s, got: %s", http.MethodGet, r.Method)
	}
	if r.UserAgent() == "" {
		t.Errorf("handler: expected UserAgent to be set, got empty string")
	}
	if r.Header.Get("Authorization") != tokenAuthScheme+" "+testToken {
		t.Errorf("handler: invalid Authorization header, want: %q, got: %q", testToken, r.Header.Get("Authorization"))
	}
}

func TestHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testCheckHeaders(t, r)
		fmt.Fprintln(w, "Hello, client")
	}))
	defer srv.Close()

	t.Log(srv.URL)

	api, err := NewAPIClient(appName, srv.URL, "abc123")
	if err != nil {
		t.Error("can't create API client:", err)
	}

	res, err := api.get("/item")
	if err != nil {
		t.Error("failed to call API endpoint:", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		t.Errorf("status code indicates failed request: expected: 2xx, got: %d", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Error("failed to read API response body:", err)
	}
	if len(body) <= 0 {
		t.Error("empty response body")
	}

	t.Logf("response: %s", body)
}
