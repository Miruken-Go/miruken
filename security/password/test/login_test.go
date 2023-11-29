package test

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/config"
	koanfp "github.com/miruken-go/miruken/config/koanf"
	"github.com/miruken-go/miruken/security/login"
	"github.com/miruken-go/miruken/security/login/callback"
	"github.com/miruken-go/miruken/security/password"
	"github.com/miruken-go/miruken/security/principal"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

type LoginTestSuite struct {
	suite.Suite
}

func (suite *LoginTestSuite) TestLogin() {
	suite.Run("JSON", func() {
		var k = koanf.New(".")
		err := k.Load(file.Provider("./login.json"), json.Parser())
		suite.Nil(err)
		handler, _ := miruken.Setup(
			config.Feature(koanfp.P(k)),
			password.Feature()).Handler()

		suite.Run("Login", func() {
			suite.Run("Succeed", func() {
				ctx := login.New("login.flow1")
				ch  := miruken.AddHandlers(
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
				ctx := login.New("login.flow1")
				ch  := miruken.AddHandlers(
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
	})

	suite.Run("ENV", func() {
		var k = koanf.New(".")
		_ = os.Setenv("Login__Flow1__0__Module", "login.pwd")
		_ = os.Setenv("Login__Flow1__0__Options__Credentials__0__Username", "user")
		_ = os.Setenv("Login__Flow1__0__Options__Credentials__0__Password", "password")
		err := k.Load(env.Provider("", "__", nil), nil,
			koanf.WithMergeFunc(koanfp.Merge))
		suite.Nil(err)
		handler, _ := miruken.Setup(
			config.Feature(koanfp.P(k)),
			password.Feature()).Handler()

		suite.Run("Login", func() {
			suite.Run("Succeed", func() {
				ctx := login.New("Login.Flow1")
				ch  := miruken.AddHandlers(
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
				ctx := login.New("Login.Flow1")
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
	})
}

func TestLoginTestSuite(t *testing.T) {
	suite.Run(t, new(LoginTestSuite))
}

