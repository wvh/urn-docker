package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleVersion(t *testing.T) {
	handler := handleVersion()

	t.Run("json", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/just-testing-man", nil)
		req.Header.Set("Accept", "application/json")
		rr := httptest.NewRecorder()
		handler(rr, req)

		res := rr.Result()

		if res.StatusCode != http.StatusOK {
			t.Errorf("handler returned wrong status code: got: %d, expected: %d", res.StatusCode, http.StatusOK)
		}

		if res.Header.Get("Content-Type") != "application/json" {
			t.Errorf("handler returned wrong content-type: got: %q, expected: %q", res.Header.Get("Content-Type"), "application/json")
		}

		versionResponse := struct {
			Id      string `json:"id"`
			Version string `json:"version"`
		}{}

		// body is an ioutil.NopCloser
		err := json.NewDecoder(res.Body).Decode(&versionResponse)
		if err != nil {
			t.Error("error decoding version body:", err)
			return
		}

		if versionResponse.Id == "" {
			t.Errorf("handler returned empty id, got: %q, expected: %s", versionResponse.Id, "string")
		}

		if versionResponse.Version == "" {
			t.Errorf("handler returned empty version, got: %q, expected: %s", versionResponse.Version, "semver")
		}
	})

	t.Run("text", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/just-testing-man", nil)
		req.Header.Set("Accept", "text/plain")
		rr := httptest.NewRecorder()
		handler(rr, req)

		res := rr.Result()

		if res.StatusCode != http.StatusOK {
			t.Errorf("handler returned wrong status code: got: %d, expected: %d", res.StatusCode, http.StatusOK)
		}

		if !strings.HasPrefix(res.Header.Get("Content-Type"), "text/plain") {
			t.Errorf("handler returned wrong content-type: got: %q, expected: %q", res.Header.Get("Content-Type"), "text/plain")
		}

		rawBody, _ := ioutil.ReadAll(res.Body)
		body := string(bytes.TrimSpace(rawBody))
		if body == "" {
			t.Errorf("handler returned empty version string, got: %q, expected: %s", body, "version string")
		}
	})
}
