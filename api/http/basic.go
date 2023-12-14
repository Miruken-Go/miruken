package http

import (
	"net/http"

	"github.com/miruken-go/miruken"
)

func BasicAuth(username, password string) Policy {
	return PolicyFunc(func(
		req *http.Request,
		composer miruken.Handler,
		next func() (*http.Response, error),
	) (*http.Response, error) {
		req.SetBasicAuth(username, password)
		return next()
	})
}
