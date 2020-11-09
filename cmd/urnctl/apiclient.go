package main

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/wvh/urn/internal/version"
)

const (
	// token authentication scheme for Authorization header (rfc7235)
	tokenAuthScheme = "Bearer"
)

type API struct {
	httpClient *http.Client
	baseURL    *url.URL
	token      string

	// cached header fields
	authHeader string
	userAgent  string
}

func NewAPIClient(app, srv, token string) (*API, error) {
	url, err := url.Parse(srv)
	if err != nil {
		return nil, fmt.Errorf("invalid API URL: %w", err)
	}
	fmt.Printf("%+v\n", url)

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: true,
		},
	}

	return &API{
		httpClient: client,
		baseURL:    url,
		token:      token,
		authHeader: tokenAuthScheme + " " + token,
		userAgent:  app + " " + version.Version,
	}, nil
}

// request sets up a basic API request for a given URL path.
// The request will add hostname and authentication information from the API client.
func (api *API) request(path string) *http.Request {
	url, err := api.baseURL.Parse(path)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Add("If-None-Match", `W/"wyzzy"`)
	req.Header.Add("Authorization", api.authHeader)
	req.Header.Add("User-Agent", api.userAgent)
	return req
}

func (api *API) get(url string) (*http.Response, error) {
	//resp, err := client.Do(req)
	return api.httpClient.Do(api.request(url))
}

func (api *API) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	//	err = json.NewDecoder(resp.Body).Decode(v)
	return resp, err
}

func (api *API) Version() {}
