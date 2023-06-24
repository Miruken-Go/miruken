package jwt

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/security"
	"github.com/miruken-go/miruken/security/claim"
	"github.com/miruken-go/miruken/security/login/callback"
	"github.com/miruken-go/miruken/security/principal"
	"math/big"
)

type (
	Token string

	LoginModule struct {
		token  Token
		id     principal.Id
		claims claim.Map
		opts   map[string]any
	}
)

var (
	ErrMissingToken = errors.New("missing security token")
	ErrEmptyToken   = errors.New("empty security token")
	ErrInvalidToken = errors.New("invalid security token")
)


func (l *LoginModule) Constructor(
	_*struct{creates.It `key:"jwt"`},
) {
}

func (l *LoginModule) Init(opts map[string]any) error {
	l.opts = opts
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

	t, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwt.ParseRSAPublicKeyFromPEM([]byte("-----BEGIN RSA PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAtVKUtcx/n9rt5afY/2WF\nNvU6PlFMggCatsZ3l4RjKxH0jgdLq6CScb0P3ZGXYbPzXvmmLiWZizpb+h0qup5j\nznOvOr+Dhw9908584BSgC83YacjWNqEK3urxhyE2jWjwRm2N95WGgb5mzE5XmZIv\nkvyXnn7X8dvgFPF5QwIngGsDG8LyHuJWlaDhr/EPLMW4wHvH0zZCuRMARIJmmqiM\ny3VD4ftq4nS5s8vJL0pVSrkuNojtokp84AtkADCDU/BUhrc2sIgfnvZ03koCQRoZ\nmWiHu86SuJZYkDFstVTVSR0hiXudFlfQ2rOhPlpObmku68lXw+7V+P7jwrQRFfQV\nXwIDAQAB\n-----END RSA PUBLIC KEY-----"))
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := t.Claims.(jwt.MapClaims); ok && t.Valid {
		l.token  = Token(tokenStr)
		l.claims = claim.Map(claims)
		if sub, err := claims.GetSubject(); err != nil && sub != "" {
			l.id = principal.Id(sub)
			subject.AddPrincipals(l.id)
		}
		subject.AddCredentials(l.token, l.claims)
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
	subject.RemoveCredentials(l.claims)
	return nil, nil
}


// pemFromExponentModule creates a public key from an exponent and modulus.
// https://baptistout.net/posts/convert-jwks-modulus-exponent-to-pem-or-ssh-public-key/
func exponentModulusToPEM(modulus, exp string) ([]byte, error) {
	// decode the base64 bytes for modulus
	nb, err := base64.RawURLEncoding.DecodeString(modulus)
	if err != nil {
		return nil, err
	}

	// The default exponent is usually 65537, so just compare the
	// base64 for [1,0,1] or [0,1,0,1]
	e := 65537
	if exp != "AQAB" && exp != "AAEAAQ" {
		eb, err := base64.RawURLEncoding.DecodeString(exp)
		if err != nil {
			return nil, err
		}
		e = int(big.NewInt(0).SetBytes(eb).Uint64())
	}

	pk := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nb),
		E: e,
	}

	der, err := x509.MarshalPKIXPublicKey(pk)
	if err != nil {
		fmt.Println(err)
	}

	block := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: der,
	}

	var out bytes.Buffer
	if err := pem.Encode(&out, block); err != nil {
		return nil, err
	} else {
		return out.Bytes(), nil
	}
}