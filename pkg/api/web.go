package api

import (
	"net/http"
)

type API struct {}

func New() (*API, error) {
	return &API{}, nil
}

func (api *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.writeHeaders(w)
	w.Write([]byte("API"))
}

func (api *API) writeHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}
