package test

import (
	"testing"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/security/login"
	"github.com/miruken-go/miruken/security/login/callback"
	"github.com/miruken-go/miruken/security/password"
	"github.com/miruken-go/miruken/security/principal"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

type LoginTestSuite struct {
	suite.Suite
}

func (suite *LoginTestSuite) TestLogin() {
	handler, _ := setup.New(password.Feature()).Context()

	flow := func() *login.Context {
		return login.NewFlow(login.Flow{{
			Module: "login.pwd",
			Options: map[string]any{
				"credentials": map[string]any{
					"user": "password",
				},
			},
		}})
	}

	suite.Run("Login", func() {
		suite.Run("Succeed", func() {
			ctx := flow()
			ch := miruken.AddHandlers(
				callback.NameHandler{Name: "user"},
				callback.PasswordHandler{Password: []byte("password")},
			)
			ps := ctx.Login(miruken.AddHandlers(handler, ch))
			suite.NotNil(ps)
			sub, err := ps.Await()
			suite.Nil(err)
			suite.NotNil(sub)
			suite.True(principal.All(sub, principal.User("user")))
			suite.Len(sub.Credentials(), 1)

			ps = ctx.Logout(handler)
			suite.NotNil(ps)
			sub, err = ps.Await()
			suite.Nil(err)
			suite.NotNil(sub)
			suite.Empty(sub.Principals())
			suite.Empty(sub.Credentials())
		})

		suite.Run("Fail", func() {
			ctx := flow()
			ch := miruken.AddHandlers(
				callback.NameHandler{Name: "user"},
				callback.PasswordHandler{Password: []byte("foo")},
			)
			ps := ctx.Login(miruken.AddHandlers(handler, ch))
			suite.NotNil(ps)
			sub, err := ps.Await()
			suite.Nil(err)
			suite.NotNil(sub)
			suite.Len(sub.Principals(), 0)
			suite.Len(sub.Credentials(), 0)
		})
	})
}

func TestLoginTestSuite(t *testing.T) {
	suite.Run(t, new(LoginTestSuite))
}
