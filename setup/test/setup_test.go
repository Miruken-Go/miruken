package test

import (
	"context"
	"errors"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

//go:generate $GOPATH/bin/miruken -tests -suffix +Bootstrap

type (
	Foo struct{}
	Bar struct{}
	Baz struct{}
)


type MultiHandler struct {
	foo Foo
	bar Bar
}

func (h *MultiHandler) HandleFoo(
	_ *handles.It, foo *Foo,
	composer miruken.Handler,
) error {
	composer.Handle(new(Bar), false, nil)
	return nil
}

func (h *MultiHandler) HandleBar(
	_ *handles.It, bar *Bar,
) miruken.HandleResult {
	return miruken.Handled
}


type EverythingHandler struct{}

func (h *EverythingHandler) HandleEverything(
	_ *handles.It, callback any,
) miruken.HandleResult {
	switch callback.(type) {
	case *Foo:
		return miruken.Handled
	default:
		return miruken.NotHandled
	}
}


type MyBootstrap struct {
}

func (b *MyBootstrap) Startup(
	ctx context.Context,
) *promise.Promise[struct{}] {
	return promise.Then(promise.Delay(ctx, 5*time.Millisecond),
		func(any) struct {} {
			return struct{}{}
		})
}

func (b *MyBootstrap) Shutdown(
	ctx context.Context,
) *promise.Promise[struct{}] {
	return promise.Then(promise.Delay(ctx, 5*time.Millisecond),
		func(any) struct {} {
			return struct{}{}
		})
}


type MyInstaller struct {
	count int
}

func (i *MyInstaller) Install(
	b *setup.Builder,
) error {
	if b.Tag(reflect.TypeOf(i)) {
		i.count++
		b.Specs(&MultiHandler{})
	}
	return nil
}


type RootInstaller struct{}

func (i *RootInstaller) DependsOn() []setup.Feature {
	return []setup.Feature{&MyInstaller{}}
}

func (i *RootInstaller) Install(
	setup *setup.Builder,
) error {
	return nil
}


type BadInstaller struct{}

func (i BadInstaller) Install(
	*setup.Builder,
) error {
	return errors.New("insufficient resources")
}

func (i BadInstaller) AfterInstall(
	*setup.Builder, miruken.Handler,
) error {
	return errors.New("process failed to start")
}


type SetupTestSuite struct {
	suite.Suite
}

func (suite *SetupTestSuite) TestSetup() {
	suite.Run("Specs", func() {
		ctx, _ := setup.New(TestFeature).Context()
		defer ctx.End(nil)

		result := ctx.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)

		result = ctx.Handle(&Baz{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.NotHandled, result)
	})

	suite.Run("ExcludeSpecs", func() {
		ctx, _ := setup.New(TestFeature).ExcludeSpecs(
			func(spec miruken.HandlerSpec) bool {
				switch ts := spec.(type) {
				case miruken.TypeSpec:
					name := ts.Name()
					return name == "MultiHandler" || strings.Contains(name, "Invalid")
				default:
					return false
				}
			},
			func(spec miruken.HandlerSpec) bool {
				if ts, ok := spec.(miruken.TypeSpec); ok {
					return ts.Type() == internal.TypeOf[*EverythingHandler]()
				}
				return false
			}).Context()
		defer ctx.End(nil)

		m, _, ok, err := miruken.Resolve[*MultiHandler](ctx)
		suite.False(ok)
		suite.Nil(err)
		suite.Nil(m)

		e, _, ok, err := miruken.Resolve[*EverythingHandler](ctx)
		suite.False(ok)
		suite.Nil(err)
		suite.Nil(e)
	})

	suite.Run("WithoutInference", func() {
		ctx, _ := setup.New(TestFeature).
			WithoutInference().
			Context()
		defer ctx.End(nil)

		result := ctx.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.NotHandled, result)

		m, _, ok, err := miruken.Resolve[*MultiHandler](ctx)
		suite.False(ok)
		suite.Nil(err)
		suite.Nil(m)
	})

	suite.Run("Installs once", func() {
		installer := &MyInstaller{}
		ctx, err := setup.New(installer, installer).Context()
		defer ctx.End(nil)
		suite.Nil(err)
		suite.Equal(1, installer.count)
		result := ctx.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
	})

	suite.Run("Installs Dependencies", func() {
		ctx, err := setup.New(&RootInstaller{}).Context()
		defer ctx.End(nil)
		suite.Nil(err)
		result := ctx.Handle(&Foo{}, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
		multi, _, ok, err := miruken.Resolve[*MultiHandler](ctx)
		suite.True(ok)
		suite.Nil(err)
		suite.NotNil(multi)
	})

	suite.Run("Overrides Dependencies", func() {
		installer := &MyInstaller{10}
		ctx, err := setup.New(&RootInstaller{}, installer).Context()
		defer ctx.End(nil)
		suite.Nil(err)
		suite.NotNil(ctx)
		suite.Equal(11, installer.count)
	})

	suite.Run("Errors", func() {
		installer := BadInstaller{}
		_, err := setup.New(installer).Context()
		suite.Equal("2 errors occurred:\n\t* insufficient resources\n\t* process failed to start\n\n", err.Error())
	})
}

func TestSetupTestSuite(t *testing.T) {
	suite.Run(t, new(SetupTestSuite))
}
