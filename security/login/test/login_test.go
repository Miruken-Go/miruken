package test

import (
	"errors"
	"fmt"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/config"
	koanfp "github.com/miruken-go/miruken/config/koanf"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/security"
	"github.com/miruken-go/miruken/security/login"
	"github.com/miruken-go/miruken/security/login/callback"
	"github.com/miruken-go/miruken/security/principal"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
	"os"
	"strconv"
	"testing"
)

type (
	MyCallbackHandler struct {
		name     string
		password []byte
	}

	MyLoginModule struct {
		user  principal.User
		debug bool
	}

	FailLoginModule struct {}
)


// MyCallbackHandler

func (h *MyCallbackHandler) Handle(
	c        any,
	greedy   bool,
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
	_*struct{creates.It `key:"my"`},
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
	_*struct{creates.It `key:"fail"`},
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
			ch  := &MyCallbackHandler{"user", []byte("1234")}
			ps  := ctx.Login(miruken.AddHandlers(handler, ch))
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
			ps  := ctx.Login(miruken.AddHandlers(handler, ch))
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
		suite.Run("File", func() {
			var k = koanf.New(".")
			err := k.Load(file.Provider("./login.json"), json.Parser())
			suite.Nil(err)
			handler, _ := setup.New(config.Feature(koanfp.P(k))).Context()
			cfg, _, ok, err := provides.Type[login.Configuration](handler, &config.Load{Path: "login"})
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(cfg)
			suite.Len(cfg, 1)
			suite.NotNil(cfg["flow1"])
			suite.Len(cfg["flow1"], 1)
			suite.Equal("module1", cfg["flow1"][0].Module)
			suite.Equal(map[string]any{
				"debug": true,
			}, cfg["flow1"][0].Options)

			f, _, ok, err := provides.Type[login.Flow](handler, &config.Load{Path: "login.flow1"})
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(f)
			suite.Equal("module1", f[0].Module)
			suite.Equal(map[string]any{
				"debug": true,
			}, f[0].Options)
		})

		suite.Run("Env", func() {
			var k = koanf.New(".")
			_ = os.Setenv("Login__Flow1__0__Module", "module1")
			_ = os.Setenv("Login__Flow1__0__Options__Debug", "true")
			err := k.Load(env.Provider("Login", "__", nil), nil,
				koanf.WithMergeFunc(koanfp.Merge))
			suite.Nil(err)
			handler, _ := setup.New(config.Feature(koanfp.P(k))).Context()
			cfg, _, ok, err := provides.Type[login.Configuration](handler, &config.Load{Path: "Login"})
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(cfg)
			suite.Len(cfg, 1)
			suite.NotNil(cfg["Flow1"])
			suite.Len(cfg["Flow1"], 1)
			suite.Equal("module1", cfg["Flow1"][0].Module)
			suite.Equal(map[string]any{
				"Debug": "true",
			}, cfg["Flow1"][0].Options)

			f, _, ok, err := provides.Type[login.Flow](handler, &config.Load{Path: "Login.Flow1"})
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(f)
			suite.Equal("module1", f[0].Module)
			suite.Equal(map[string]any{
				"Debug": "true",
			}, f[0].Options)
		})

		suite.Run("No Modules", func() {
			var k = koanf.New(".")
			err := k.Load(env.Provider("Login", "__", nil), nil,
				koanf.WithMergeFunc(koanfp.Merge))
			suite.Nil(err)
			handler, _ := setup.New(config.Feature(koanfp.P(k))).Context()
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
