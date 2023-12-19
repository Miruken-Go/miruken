package auth

import (
	"net/http"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/security/login/callback"
)

// Basic is a http authentication Scheme that uses a
// username and password to protect resources.
type Basic struct {
	Realm string
}

func (b Basic) Accept(
	r *http.Request,
) (miruken.Handler, error, bool) {
	if user, pass, ok := r.BasicAuth(); ok {
		return miruken.AddHandlers(
			callback.NameHandler{Name: user},
			callback.PasswordHandler{Password: []byte(pass)},
		), nil, true
	}
	return nil, nil, false
}

func (b Basic) Challenge(
	w   http.ResponseWriter,
	r   *http.Request,
	err error,
) int {
	WriteWWWAuthenticateHeader(w, "Bearer", b.Realm, nil, err)
	return http.StatusUnauthorized
}

// Basic configures an authentication flow to use `Basic` auth.
func (b *FlowBuilder) Basic() *Authentication {
	return b.Scheme(Basic{})
}

// BasicInRealm configures an authentication flow to use `Basic`
// auth in the supplied realm.
func (b *FlowBuilder) BasicInRealm(realm string) *Authentication {
	return b.Scheme(Basic{realm})
}
