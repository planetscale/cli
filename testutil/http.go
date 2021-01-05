package testutil

import (
	"net/http"
	"net/http/httptest"
)

func SetupServer(fn func(mux *http.ServeMux)) (*httptest.Server, func()) {
	mux := http.NewServeMux()

	fn(mux)
	server := httptest.NewServer(mux)

	return server, func() {
		server.Close()
	}
}
