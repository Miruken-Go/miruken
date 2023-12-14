package jwt

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/security"
	"github.com/miruken-go/miruken/security/login/callback"
	"github.com/miruken-go/miruken/security/principal"
)

type (
	// LoginModule authenticates a subject from a JWT (JSON Web Token).
	LoginModule struct {
		issuer     string
		audience   string
		jwksUri    string
		jwksJson   json.RawMessage
		token      *jwt.Token
		principals []security.Principal
		jwks       KeySet
	}

	// KeySet provides JWKS (JSON Web Key Sets) to verify JWT signatures.
	KeySet interface {
		At(jwksURI string) *promise.Promise[jwt.Keyfunc]
		From(jwksJSON json.RawMessage) (jwt.Keyfunc, error)
	}

	// Scope is a grouping of claims which effectively represents
	// a permission that is set on a token. e.g. orders:read
	Scope string
)

var (
	ErrScopeNameRequired     = errors.New("scope name is required")
	ErrInvalidAudOption      = errors.New("invalid audience option")
	ErrInvalidJwksOption     = errors.New("invalid jwks option")
	ErrInvalidJwksUrl        = errors.New("invalid jwks.uri option")
	ErrJwksUrlOrKeysRequired = errors.New("option jwks.uri or jwks.keys is required")
	ErrMissingToken          = errors.New("missing security token")
	ErrEmptyToken            = errors.New("empty security token")
	ErrInvalidToken          = errors.New("invalid security token")
	ErrInvalidClaims         = errors.New("invalid security claims")
)

//goland:noinspection GoMixedReceiverTypes
func (s Scope) Name() string {
	return string(s)
}

//goland:noinspection GoMixedReceiverTypes
func (s *Scope) InitWithTag(tag reflect.StructTag) error {
	if name, ok := tag.Lookup("name"); ok {
		if name == "" {
			return ErrScopeNameRequired
		}
		*s = Scope(name)
	}
	return nil
}

func (l *LoginModule) Constructor(
	_ *struct {
		creates.It `key:"login.jwt"`
	}, jwks KeySet,
) {
	l.jwks = jwks
}

func (l *LoginModule) Init(opts map[string]any) error {
	for k, opt := range opts {
		switch strings.ToLower(k) {
		case "issuer":
			if iss, ok := opt.(string); !ok {
				return ErrInvalidAudOption
			} else {
				l.issuer = iss
			}
		case "audience":
			if aud, ok := opt.(string); !ok {
				return ErrInvalidAudOption
			} else {
				l.audience = aud
			}
		case "jwks":
			if jwks, ok := opt.(map[string]any); !ok {
				return ErrInvalidJwksOption
			} else {
				for jk, jv := range jwks {
					switch strings.ToLower(jk) {
					case "uri":
						if uri, ok := jv.(string); !ok {
							return ErrInvalidJwksUrl
						} else {
							l.jwksUri = uri
						}
					case "keys":
						keys := map[string]any{"keys": jv}
						if js, err := json.Marshal(keys); err != nil {
							return err
						} else {
							l.jwksJson = js
						}
					}
				}
			}
		}
	}
	if (l.jwksUri == "") == (len(l.jwksJson) == 0) {
		return ErrJwksUrlOrKeysRequired
	}
	return nil
}

func (l *LoginModule) Login(
	subject security.Subject,
	handler miruken.Handler,
) error {
	name := callback.NewName("prompt", "")
	if !handler.Handle(name, false, nil).Handled() {
		return ErrMissingToken
	}
	tokenStr := name.Name()
	if tokenStr == "" {
		return ErrEmptyToken
	}

	keys, err := l.keys()
	if err != nil {
		return err
	}

	token, err := jwt.Parse(tokenStr, keys,
		jwt.WithIssuer(l.issuer), jwt.WithAudience(l.audience))
	if err != nil {
		return err
	} else if !token.Valid {
		return ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		l.token = token
		if sub, err := claims.GetSubject(); err != nil && sub != "" {
			id := principal.Id(sub)
			subject.AddPrincipals(id)
			l.principals = append(l.principals, id)
		}
		subject.AddCredentials(l.token)
		l.addScopes(subject, claims)
		l.addKnownPrincipals(subject, claims)
	} else {
		return ErrInvalidClaims
	}

	return nil
}

func (l *LoginModule) Logout(
	subject security.Subject,
	handler miruken.Handler,
) error {
	subject.RemovePrincipals(l.principals...)
	subject.RemoveCredentials(l.token)
	l.token = nil
	return nil
}

func (l *LoginModule) keys() (k jwt.Keyfunc, err error) {
	if ks := l.jwksJson; len(ks) > 0 {
		k, err = l.jwks.From(ks)
	} else {
		k, err = l.jwks.At(l.jwksUri).Await()
	}
	return
}

func (l *LoginModule) addScopes(
	subject security.Subject,
	claims jwt.MapClaims,
) {
	if scp, ok := claims["scp"]; ok {
		scopes := strings.Split(scp.(string), " ")
		for _, scope := range scopes {
			scp := Scope(scope)
			subject.AddPrincipals(scp)
			l.principals = append(l.principals, scp)
		}
	}
}

func (l *LoginModule) addKnownPrincipals(
	subject security.Subject,
	claims jwt.MapClaims,
) {
	for key, val := range claims {
		switch strings.ToLower(key) {
		case "email":
			subject.AddPrincipals(principal.Email(val.(string)))
		case "roles":
			roles := principal.Parse[principal.Role](val)
			subject.AddPrincipals(roles...)
		case "groups":
			groups := principal.Parse[principal.Role](val)
			subject.AddPrincipals(groups...)
		case "entitlements":
			entitlements := principal.Parse[principal.Entitlement](val)
			subject.AddPrincipals(entitlements...)
		}
	}
}
