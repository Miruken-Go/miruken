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
	"reflect"
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
	ErrMissingToken  = errors.New("missing security token")
	ErrEmptyToken    = errors.New("empty security token")
	ErrInvalidToken  = errors.New("invalid security token")
	ErrInvalidClaims = errors.New("invalid security claims")
)

//goland:noinspection GoMixedReceiverTypes
func (s Scope) Name() string {
	return string(s)
}

//goland:noinspection GoMixedReceiverTypes
func (s *Scope) InitWithTag(tag reflect.StructTag) error {
	if name, ok := tag.Lookup("name"); ok {
		if name == "" {
			return errors.New("scope name is required")
		}
		*s = Scope(name)
	}
	return nil
}


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
		return errors.New("missing JwksUrl option")
	}
	return nil
}

func (l *LoginModule) Login(
	subject security.Subject,
	handler miruken.Handler,
) error {
	name := callback.NewName("prompt", "")
	result := handler.Handle(name, false, nil)
	if !result.Handled() {
		return ErrMissingToken
	}
	tokenStr := name.Name()
	if tokenStr == "" {
		return ErrEmptyToken
	}

	keyfunc, err := l.jwks.At(l.jwksUrl).Await()
	if err != nil {
		return err
	}

	token, err := jwt.Parse(tokenStr, keyfunc)
	if err != nil {
		return err
	} else if !token.Valid {
		return ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		l.token = token
		if sub, err := claims.GetSubject(); err != nil && sub != "" {
			l.id = principal.Id(sub)
			subject.AddPrincipals(l.id)
		}
		subject.AddCredentials(l.token)
		l.addScopes(subject, claims)
	} else {
		return ErrInvalidClaims
	}

	return nil
}

func (l *LoginModule) Logout(
	subject security.Subject,
	handler miruken.Handler,
) error {
	subject.RemovePrincipals(l.id)
	subject.RemoveCredentials(l.token)
	for _, scope := range l.scopes {
		subject.RemovePrincipals(scope)
	}
	return nil
}

func (l *LoginModule) addScopes(
	subject security.Subject,
	claims jwt.MapClaims,
) {
	if scp, ok := claims["scp"]; ok {
		scopes := strings.Split(scp.(string), " ")
		l.scopes = make([]Scope, len(scopes))
		for i, scope := range scopes {
			l.scopes[i] = Scope(scope)
			subject.AddPrincipals(l.scopes[i])
		}
	}
}