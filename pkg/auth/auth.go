package auth

import (
	"errors"
	"net/http"
	"strings"
)

const headerTokenType = "apiv1"

var (
	// ErrInvalidAuthPolicy means the auth policy could not be parsed from string.
	ErrInvalidAuthPolicy = errors.New("invalid auth policy")
)

type authPolicy int

// Auth policies defined.
const (
	Unset authPolicy = iota
	Skip
	Pass
	WriteOnly
	All
)

func (policy authPolicy) String() string {
	switch policy {
	case Unset:
		return "Unset"
	case Skip:
		return "Skip"
	case Pass:
		return "Pass"
	case WriteOnly:
		return "WriteOnly"
	case All:
		return "All"
	}
	return "Unset"
}

// Handler blah blah.
type Handler struct {
	policy authPolicy
}

// NewHandler creates an authorization handler from the provided policy constant.
func NewHandler(policy authPolicy) func(http.Handler) http.Handler {
	/*
		return &Handler{
			policy: policy,
		}
	*/
	h := &Handler{
		policy: policy,
	}

	switch policy {
	case Unset:
		return h.unauth
	case Skip:
		return func(next http.Handler) http.Handler {
			return next
		}
	case Pass:
		return h.pass
	case WriteOnly:
		return h.ro
	case All:
		return h.rw
	default:
		return h.unauth
	}

}

// NewHandlerFromString parses the given string into an auth policy, then returns the auth handler.
// If the provided string can't be parsed into an auth policy constant, an error is returned.
// This constructor can be used to setup authentication policy from the environment or other means of configuration.
//func NewHandlerFromString(s string) (*Handler, error) {
func NewHandlerFromString(s string) (func(http.Handler) http.Handler, error) {
	policy, err := fromString(s)
	if err != nil {
		return nil, err
	}
	return NewHandler(policy), nil
}

func fromString(s string) (authPolicy, error) {
	switch strings.ToLower(s) {
	case "unset":
		return Unset, nil
	case "skip":
		return Skip, nil
	case "pass":
		return Pass, nil
	case "writeonly":
		return WriteOnly, nil
	case "all":
		return All, nil
	default:
		return Unset, ErrInvalidAuthPolicy
	}
}

// parse token from request header
func (h *Handler) userFromRequest(r *http.Request) string {
	hdr := r.Header.Get("Authorization")
	if !strings.HasPrefix(hdr, headerTokenType) {
		return ""
	}

	user := parseToken(hdr[len(headerTokenType):])
	return user
}

func (h *Handler) pass(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = h.userFromRequest(r)
		next.ServeHTTP(w, r)
	})
}

// middleware that allows HTTP methods with no side effects without authentication
func (h *Handler) ro(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// authenticated, pass
		if h.userFromRequest(r) != "" {
			next.ServeHTTP(w, r)
		}

		// not authenticated, only read methods
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			next.ServeHTTP(w, r)
		default:
			h.forbidden(w, r)
		}
	})
}

// middleware that allows HTTP methods with no side effects without authentication
func (h *Handler) rw(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.userFromRequest(r) != "" {
			next.ServeHTTP(w, r)
		}
		h.forbidden(w, r)
	})
}

// middleware that returns 403 Forbidden for all requests
func (h *Handler) unauth(_ http.Handler) http.Handler {
	return http.HandlerFunc(h.forbidden)
}

// shim for whatever HTTP error function
func (h *Handler) forbidden(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
}

func parseToken(token string) string {
	return token
}
