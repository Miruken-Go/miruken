package test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/config"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/security"
	"github.com/miruken-go/miruken/security/login"
	"github.com/miruken-go/miruken/security/login/callback"
	"github.com/miruken-go/miruken/security/principal"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

// emptyProvider is a trivial config.Provider that leaves the output
// at its zero value, used to exercise config-driven flow resolution
// without depending on a real config backend.
type emptyProvider struct{}

func (emptyProvider) Unmarshal(_ string, _ bool, _ any) error {
	return nil
}

type (
	MyCallbackHandler struct {
		name     string
		password []byte
	}

	MyLoginModule struct {
		user  principal.User
		debug bool
	}

	FailLoginModule struct{}
)

// MyCallbackHandler

func (h *MyCallbackHandler) Handle(
	c any,
	greedy bool,
	composer miruken.Handler,
) miruken.HandleResult {
	switch cb := c.(type) {
	case *callback.Name:
		cb.SetName(h.name)
	case *callback.Password:
		cb.SetPassword(h.password)
	default:
		return miruken.NotHandled
	}
	return miruken.Handled
}

// MyLoginModule

func (l *MyLoginModule) Constructor(
	_ *struct {
		creates.It `key:"my"`
	},
	opts map[string]any,
) error {
	switch d := opts["debug"].(type) {
	case nil:
	case bool:
		l.debug = d
	case string:
		if debug, err := strconv.ParseBool(d); err != nil {
			return err
		} else {
			l.debug = debug
		}
	default:
		return fmt.Errorf(`unrecognized "debug" option: %v`, d)
	}
	return nil
}

func (l *MyLoginModule) Login(
	subject security.Subject,
	handler miruken.Handler,
) error {
	name := callback.NewName("user name: ", "")
	result := handler.Handle(name, false, nil)
	if !result.Handled() {
		return errors.New("username unavailable")
	} else if name.Name() != "test" {
		return errors.New("incorrect username")
	}

	password := callback.NewPassword("password: ", false)
	result = handler.Handle(password, false, nil)
	if !result.Handled() {
		return errors.New("password unavailable")
	} else if string(password.Password()) != "password" {
		return errors.New("incorrect password")
	}

	if l.debug {
		fmt.Println("\t[MyLoginModule]", "username:", name.Name())
		fmt.Println("\t[MyLoginModule]", "password:", string(password.Password()))
	}

	l.user = principal.User(name.Name())
	subject.AddPrincipals(l.user)

	return nil
}

func (l *MyLoginModule) Logout(
	subject security.Subject,
	handler miruken.Handler,
) error {
	subject.RemovePrincipals(l.user)
	return nil
}

// FailLoginModule

func (l *FailLoginModule) Constructor(
	_ *struct {
		creates.It `key:"fail"`
	},
) {
}

func (l *FailLoginModule) Login(
	subject security.Subject,
	handler miruken.Handler,
) error {
	return errors.New("idp not responding")
}

func (l *FailLoginModule) Logout(
	subject security.Subject,
	handler miruken.Handler,
) error {
	return nil
}

type LoginTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *LoginTestSuite) SetupTest() {
	suite.specs = []any{
		&MyLoginModule{},
		&FailLoginModule{},
	}
}

func (suite *LoginTestSuite) Setup(specs ...any) (*context.Context, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return setup.New().Specs(specs...).Context()
}

func (suite *LoginTestSuite) TestLogin() {
	suite.Run("Login", func() {
		suite.Run("Succeed", func() {
			handler, _ := suite.Setup()
			ctx := login.NewFlow(login.Flow{
				{Module: "my", Options: map[string]any{"debug": true}},
			})
			ch := &MyCallbackHandler{"test", []byte("password")}
			ps := ctx.Login(miruken.AddHandlers(handler, ch))
			suite.NotNil(ps)
			sub, err := ps.Await()
			suite.Nil(err)
			suite.NotNil(sub)
			suite.True(principal.All(sub, principal.User("test")))
		})

		suite.Run("Fail", func() {
			handler, _ := suite.Setup()
			ctx := login.NewFlow(login.Flow{{Module: "my"}})
			ch := &MyCallbackHandler{"user", []byte("1234")}
			ps := ctx.Login(miruken.AddHandlers(handler, ch))
			suite.NotNil(ps)
			sub, err := ps.Await()
			suite.NotNil(err)
			suite.Nil(sub)
			var le login.Error
			suite.ErrorAs(err, &le)
			suite.Equal("login failed: incorrect username", le.Error())
		})

		suite.Run("Recover", func() {
			handler, _ := suite.Setup()
			ctx := login.NewFlow(login.Flow{{Module: "my"}, {Module: "fail"}})
			ch := &MyCallbackHandler{"test", []byte("password")}
			ps := ctx.Login(miruken.AddHandlers(handler, ch))
			suite.NotNil(ps)
			sub, err := ps.Await()
			suite.NotNil(err)
			suite.Nil(sub)
			var le login.Error
			suite.ErrorAs(err, &le)
			suite.Equal("login failed: idp not responding", le.Error())
		})

		suite.Run("No Modules", func() {
			defer func() {
				if r := recover(); r != nil {
					suite.Equal("login: flow requires at least one module", r)
				}
			}()
			login.NewFlow(login.Flow{})
		})
	})

	suite.Run("Logout", func() {
		suite.Run("Succeed", func() {
			handler, _ := suite.Setup()
			ctx := login.NewFlow(login.Flow{
				{Module: "my", Options: map[string]any{"debug": true}},
			})
			ch := &MyCallbackHandler{"test", []byte("password")}
			ps := ctx.Login(miruken.AddHandlers(handler, ch))
			suite.NotNil(ps)
			sub, err := ps.Await()
			suite.Nil(err)
			suite.NotNil(sub)
			suite.True(principal.All(sub, principal.User("test")))

			ps = ctx.Logout(handler)
			suite.NotNil(ps)
			sub, err = ps.Await()
			suite.Nil(err)
			suite.NotNil(sub)
			suite.Empty(sub.Principals())
		})

		suite.Run("Login Required", func() {
			handler, _ := suite.Setup()
			ctx := login.NewFlow(login.Flow{
				{Module: "my", Options: map[string]any{"debug": true}},
			})
			ps := ctx.Logout(handler)
			suite.NotNil(ps)
			_, err := ps.Await()
			suite.NotNil(err)
			var le login.Error
			suite.ErrorAs(err, &le)
			suite.Equal(`login failed: login must succeed first`, le.Error())
		})
	})

	suite.Run("Configuration", func() {
		suite.Run("No Modules", func() {
			handler, _ := setup.New(config.Feature(emptyProvider{})).Context()
			ctx := login.New("login.flow")
			ps := ctx.Login(handler)
			suite.NotNil(ps)
			sub, err := ps.Await()
			suite.NotNil(err)
			suite.Nil(sub)
			var le login.Error
			suite.ErrorAs(err, &le)
			suite.Equal("login failed: config: flow requires at least one module", le.Error())
		})
	})
}

func TestLoginTestSuite(t *testing.T) {
	suite.Run(t, new(LoginTestSuite))
}
