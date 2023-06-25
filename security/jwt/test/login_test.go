package test

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/security/jwt"
	"github.com/miruken-go/miruken/security/jwt/jwks"
	"github.com/miruken-go/miruken/security/login"
	"github.com/miruken-go/miruken/security/login/callback"
	"github.com/stretchr/testify/suite"
	"testing"
)

type (
	TokenCallbackHandler struct {
		token string
	}
)


// TokenCallbackHandler

func (h *TokenCallbackHandler) Handle(
	c        any,
	greedy   bool,
	composer miruken.Handler,
) miruken.HandleResult {
	switch cb := c.(type) {
	case *callback.Name:
		cb.SetName(h.token)
	default:
		return miruken.NotHandled
	}
	return miruken.Handled
}


type LoginTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *LoginTestSuite) SetupTest() {
	suite.specs = []any{
	}
}

func (suite *LoginTestSuite) Setup(specs ...any) (miruken.Handler, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return miruken.Setup(jwt.Feature(), jwks.Feature()).
		Specs(specs...).Handler()
}

func (suite *LoginTestSuite) TestLogin() {
	suite.Run("Login", func() {
		handler, _ := suite.Setup()
		ctx := login.New(login.ModuleEntry{Module: "jwt", Options: map[string]any{
			"JWKSUrl": "https://teamsrvdevcraig.b2clogin.com/teamsrvdevcraig.onmicrosoft.com/b2c_1_signin/discovery/v2.0/keys",
		}})
		ch := &TokenCallbackHandler{"eyJhbGciOiJSUzI1NiIsImtpZCI6Ilg1ZVhrNHh5b2pORnVtMWtsMll0djhkbE5QNC1jNTdkTzZRR1RWQndhTmsiLCJ0eXAiOiJKV1QifQ.eyJvaWQiOiI4ZTExNzQ1Yi1jZDdlLTQzZTctOTRjNS02MzdmMzJhZGU5NDgiLCJzdWIiOiI4ZTExNzQ1Yi1jZDdlLTQzZTctOTRjNS02MzdmMzJhZGU5NDgiLCJnaXZlbl9uYW1lIjoiQ3JhaWciLCJmYW1pbHlfbmFtZSI6Ik5ldXdpcnQiLCJlbWFpbHMiOlsiY25ldXdpcnRAZ21haWwuY29tIl0sInRmcCI6IkIyQ18xX3NpZ25pbiIsInNjcCI6IlRlYW0uQ3JlYXRlIFBlcnNvbi5DcmVhdGUiLCJhenAiOiJhZWIwMTBjNi1iZjNmLTQxMTMtOGVmZi05NDRjNDg3MmVhNmEiLCJ2ZXIiOiIxLjAiLCJpYXQiOjE2ODc1NTIzMTcsImF1ZCI6IjYwZjEyM2FiLWRlNGQtNGQyZi1iYjkzLWI1NGZkZGMzOGVlMSIsImV4cCI6MTY4NzU1NTkxNywiaXNzIjoiaHR0cHM6Ly90ZWFtc3J2ZGV2Y3JhaWcuYjJjbG9naW4uY29tLzA0OGNmMjA4LTc3OGYtNDk2Yi1iODkyLTlkMDNkMTU2NTJjZC92Mi4wLyIsIm5iZiI6MTY4NzU1MjMxN30.ZxR1ZflO5GDyhbVLDzg67k2v-rb_ns9jbVYhkFLxsaPimUyOrtqMMCPP9vRpQ70YJQQrSo8BaW5Qkorf6wYHjqTAKegA8ucLopWOdmiklRBoPQKdBCQhC7k6S4Kin7YhGwenJiVhJtcL4BLOV8GLXiT_mTj6tpMak6JrzPaJWf-yTW0CqsaLLgoowyCFD4AJEFe0ri_z-uGaZ99zlHWYWkEzcijQt1iEeQRZQbnE-eFPYafrBRzEwMUI1WVf50p7NpVnV79wLlvor7adEJzFEeOP_udXdo6fPKK2QPGDE0bagEzzleOYFoREqlhP61-BW8-iY2t57Y9tenhOBWdQAA"}
		ps := ctx.Login(miruken.AddHandlers(handler, ch))
		suite.NotNil(ps)
		_, err := ps.Await()
		suite.NotNil(err)
		var le login.Error
		suite.ErrorAs(err, &le)
		suite.Equal("login failed: token has invalid claims: token is expired", le.Error())
	})
}

func TestLoginTestSuite(t *testing.T) {
	suite.Run(t, new(LoginTestSuite))
}
