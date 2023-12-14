package auth

import (
	"net/http"
	"strings"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/security/login/callback"
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
	w http.ResponseWriter,
	r *http.Request,
	err error,
) int {
	WriteWWWAuthenticateHeader(w, "Bearer", b.Realm, nil, err)
	return http.StatusUnauthorized
}

// Bearer configures an authentication flow to use `Bearer` tokens.
func (b *FlowBuilder) Bearer() *Authentication {
	return b.Scheme(Bearer{})
}

// BearerInRealm configures an authentication flow to use `Bearer`
// tokens in the supplied realm.
func (b *FlowBuilder) BearerInRealm(realm string) *Authentication {
	return b.Scheme(Bearer{realm})
}
