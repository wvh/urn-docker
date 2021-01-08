package auth

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testBody = "Well hello there!"

func hasBody(res *http.Response, s string) bool {
	// body is bytes.Buffer, no close required
	body, _ := ioutil.ReadAll(res.Body)
	return strings.TrimSpace(string(body)) == s
}

func isSuccess(status int) bool {
	return status >= 200 && status < 300
}

type HasBeenCalledHandler bool

func (h *HasBeenCalledHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	*h = true
	io.WriteString(w, testBody)
}

func TestAuth(t *testing.T) {
	/*
		endpoint := func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, testBody)
		}
	*/
	validToken := "jack"

	tests := []struct {
		name    string
		method  string
		handler func(http.Handler) http.Handler
		status  int
		token   string
	}{
		// read method without token
		{"none", http.MethodGet, nil, 200, ""},
		{"unset", http.MethodGet, NewHandler(Unset), 403, ""},
		{"skip", http.MethodGet, NewHandler(Skip), 200, ""},
		{"pass", http.MethodGet, NewHandler(Pass), 200, ""},
		{"writeonly", http.MethodGet, NewHandler(WriteOnly), 200, ""},
		{"all", http.MethodGet, NewHandler(All), 403, ""},
		// write method without token
		{"none", http.MethodPost, nil, 200, ""},
		{"unset", http.MethodPost, NewHandler(Unset), 403, ""},
		{"skip", http.MethodPost, NewHandler(Skip), 200, ""},
		{"pass", http.MethodPost, NewHandler(Pass), 200, ""},
		{"writeonly", http.MethodPost, NewHandler(WriteOnly), 403, ""},
		{"all", http.MethodPost, NewHandler(All), 403, ""},
		// write method with token
		{"none", http.MethodPut, nil, 200, validToken},
		{"unset", http.MethodPut, NewHandler(Unset), 403, validToken},
		{"skip", http.MethodPut, NewHandler(Skip), 200, validToken},
		{"pass", http.MethodPut, NewHandler(Pass), 200, validToken},
		{"writeonly", http.MethodPut, NewHandler(WriteOnly), 200, validToken},
		{"all", http.MethodPut, NewHandler(All), 200, validToken},
		// invalid policy
		{"pass", http.MethodGet, NewHandler(-1), 403, validToken},
		// from string
		{"pass", http.MethodGet, func() func(http.Handler) http.Handler { h, _ := NewHandlerFromString("pass"); return h }(), 200, ""},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %s%s", test.name, test.method, func(s string) string {
			if s != "" {
				return " token"
			}
			return ""
		}(test.token)), func(t *testing.T) {
			endpoint := new(HasBeenCalledHandler)
			var handler http.Handler

			if test.handler != nil {
				handler = test.handler(endpoint)
			} else {
				handler = endpoint
			}

			req := httptest.NewRequest(test.method, "http://example.com/api/meh", nil)
			if test.token != "" {
				req.Header.Add("Authorization", headerTokenType+" "+test.token)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			res := w.Result()

			if res.StatusCode != test.status {
				t.Errorf("response to %s request has wrong status code, want: %d, got: %d", test.method, test.status, res.StatusCode)
			}

			if isSuccess(test.status) && !bool(*endpoint) {
				t.Errorf("success status code but endpoint was not called, status: %d, called: %t", test.status, *endpoint)
			}

			if !isSuccess(test.status) && bool(*endpoint) {
				t.Errorf("endpoint was called but should not have been, status: %d, called: %t", test.status, *endpoint)
			}

			/*
				if isSuccess(res.StatusCode) && !hasBody(res, testBody) {
					t.Errorf("body of response differs from test body, want: %s, got: %s", testBody, res.Body)
				}
			*/
		})
	}
}

func TestParseToken(t *testing.T) {
	tokenFunc := parseToken
	tokens := []struct {
		enc     string
		payload string
		valid   bool
	}{
		{"jack", "jack", true},
	}

	for _, token := range tokens {
		t.Run(token.enc, func(t *testing.T) {
			payload := tokenFunc(token.enc)

			if token.valid && payload == "" {
				t.Errorf("valid token should return payload, want: %q, got: %q", token.payload, payload)
			}

			if !token.valid && payload != "" {
				t.Errorf("invalid token should not return payload, want: %q, got: %q", token.payload, payload)
			}

			if payload != token.payload {
				t.Errorf("payload doesn't match, want: %q, got: %q", token.payload, payload)
			}
		})
	}
}

func TestFromString(t *testing.T) {
	tests := []struct {
		s   string
		pol authPolicy
		err error
	}{
		{"unset", Unset, nil},
		{"SKIP", Skip, nil},
		{"pASS", Pass, nil},
		{"writeOnly", WriteOnly, nil},
		{"ALL", All, nil},

		{"höpö", Unset, ErrInvalidAuthPolicy},
		{"unset", -1, nil},
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			pol, err := fromString(test.s)

			if test.err == nil && err != nil {
				t.Error("unexpected error:", err)
			}

			if test.err != nil && err == nil {
				t.Errorf("test should return error, want: %v, got: %v", test.err, err)
			}

			// we use negative values as sentinels to test non-existing policies
			if pol != test.pol && test.pol >= 0 {
				t.Errorf("wrong policy, want: %s, got: %s", test.pol, pol)
			}
		})

		t.Run(test.pol.String(), func(t *testing.T) {
			if test.err != nil {
				return
			}
			if strings.ToLower(test.pol.String()) != strings.ToLower(test.s) {
				t.Errorf("wrong string for policy, want: %q, got: %q", strings.ToLower(test.s), strings.ToLower(test.pol.String()))
			}
		})
	}
}

func TestInvalidAuthPolicyError(t *testing.T) {
	_, err := NewHandlerFromString("passata di pomodoro")
	if err != ErrInvalidAuthPolicy {
		t.Errorf("unexpected error, want: %v, got: %v", ErrInvalidAuthPolicy, err)
	}
}
