package http

import (
	"github.com/miruken-go/miruken"
	"net/http"
)

func BasicAuth(username, password string) Policy {
	return func(
		req      *http.Request,
		composer miruken.Handler,
		next     func() (*http.Response, error),
	)  (*http.Response, error) {
		req.SetBasicAuth(username, password)
		return next()
	}
}
