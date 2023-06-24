package jwt

import (
	"encoding/json"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/security"
	"github.com/miruken-go/miruken/security/login/callback"
	"github.com/miruken-go/miruken/security/principal"
	"strings"
)

type (
	// LoginModule authenticates a subject from a JWT (JSON Web Token).
	LoginModule struct {
		token   *jwt.Token
		id      principal.Id
		scopes  []Scope
		jwksUrl string
		jwks    KeySet
	}

	// KeySet provides JWKS (JSON Web Key Sets) to verify JWT signatures.
	KeySet interface {
		At(jwksURL string) *promise.Promise[jwt.Keyfunc]
		From(jwksJSON json.RawMessage) (jwt.Keyfunc, error)
	}

	// Scope is a grouping of claims which effectively represents
	// a permission that is set on a token. e.g. orders:read
	Scope string
)

var (
	ErrMissingToken = errors.New("missing security token")
	ErrEmptyToken   = errors.New("empty security token")
	ErrInvalidToken = errors.New("invalid security token")
)


func (l *LoginModule) Constructor(
	_*struct{creates.It `key:"jwt"`}, jwks KeySet,
) {
	l.jwks = jwks
}

func (l *LoginModule) Init(opts map[string]any) error {
	for k,opt := range opts {
		switch strings.ToUpper(k) {
		case "JWKSURL":
			l.jwksUrl = opt.(string)
		}
	}
	if l.jwksUrl == "" {
		return errors.New("missing JWKSUrl option")
	}
	return nil
}

func (l *LoginModule) Login(
	subject security.Subject,
	handler miruken.Handler,
) (*promise.Promise[any], error) {
	name := callback.NewName("prompt", "")
	result := handler.Handle(name, false, nil)
	if !result.Handled() {
		return nil, ErrMissingToken
	}
	tokenStr := name.Name()
	if tokenStr == "" {
		return nil, ErrEmptyToken
	}

	keyfunc, err := l.jwks.At(l.jwksUrl).Await()
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenStr, keyfunc)
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		l.token = token
		if sub, err := claims.GetSubject(); err != nil && sub != "" {
			l.id = principal.Id(sub)
			subject.AddPrincipals(l.id)
		}
		subject.AddCredentials(l.token)
	} else {
		return nil, ErrInvalidToken
	}

	return nil, nil
}

func (l *LoginModule) Logout(
	subject security.Subject,
	handler miruken.Handler,
) (*promise.Promise[any], error) {
	subject.RemovePrincipals(l.id)
	subject.RemoveCredentials(l.token)
	for _, scope := range l.scopes {
		subject.RemovePrincipals(scope)
	}
	return nil, nil
}
