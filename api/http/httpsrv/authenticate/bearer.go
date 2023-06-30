package authenticate

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/security/login/callback"
	"net/http"
	"strings"
)

// Bearer is a http authentication Scheme that uses an
// opaque string (token) to protect resources.
type Bearer struct {
	Realm string
}


func (b Bearer) Accept(
	r *http.Request,
) (miruken.Handler, error, bool) {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return callback.NameHandler{Name: auth[7:]}, nil, true
	}
	return nil, nil, false
}

func (b Bearer) Challenge(
	w   http.ResponseWriter,
	r   *http.Request,
	err error,
) int {
	h := "Bearer"
	if realm := b.Realm; realm != "" {
		h = "Bearer realm=\"" + realm + "\""
	}
	w.Header().Set("WWW-Authenticate", h)
	return http.StatusUnauthorized
}

func (b Bearer) Forbid(
	w   http.ResponseWriter,
	r   *http.Request,
	err error,
) {
	w.WriteHeader(http.StatusForbidden)
}


func (b *FlowBuilder) Bearer() *Authentication {
	return b.Scheme(Bearer{})
}