package test

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/suite"
	miruken "miruken.com/miruken"
	"reflect"
	"strings"
	"testing"
)

//go:generate $GOPATH/bin/mirukentypes -tests

type Counter interface {
	Count() int
	Inc() int
}

type Counted struct {
	count int
}

func (c *Counted) Count() int {
	return c.count
}

func (c *Counted) Inc() int {
	c.count++
	return c.count
}

type (
	Foo struct { Counted }
	Bar struct { Counted }
	Baz struct { Counted }
	Bam struct { Counted }
	Boo struct { Counted }
)

// FooHandler
type FooHandler struct{}

func (h *FooHandler) Handle(
	callback interface{},
	greedy   bool,
	composer miruken.Handler,
) miruken.HandleResult {
	switch foo := callback.(type) {
	case *Foo:
		foo.Inc()
		return composer.Handle(Bar{}, false, nil)
	default:
		return miruken.NotHandled
	}
}

// BarHandler
type BarHandler struct {}

func (h *BarHandler) HandleBar(
	_ miruken.Handles,
	bar Bar,
) {
}

// CounterHandler
type CounterHandler struct {}

func (h *CounterHandler) HandleCounted(
	_ miruken.Handles,
	counter Counter,
) (Counter, miruken.HandleResult) {
	switch c := counter.Inc(); {
	case c % 3 == 0:
		err := fmt.Errorf("%v is divisible by 3", c)
		return nil, miruken.NotHandled.WithError(err)
	case c % 2 == 0: return nil, miruken.NotHandled
	default: return counter, miruken.Handled
	}
}

// MultiHandler
type MultiHandler struct {
	foo Foo
	bar Bar
}

func (h *MultiHandler) HandleFoo(
	_ miruken.Handles,
	foo      *Foo,
	composer miruken.Handler,
) error {
	h.foo.Inc()
	if foo.Inc() == 5 {
		return errors.New("count reached 5")
	}
	composer.Handle(new(Bar), false, nil)
	return nil
}

func (h *MultiHandler) HandleBar(
	_ miruken.Handles,
	bar *Bar,
) miruken.HandleResult {
	h.bar.Inc()
	if bar.Inc() % 2 == 0 {
		return miruken.Handled
	}
	return miruken.NotHandled
}

// EverythingHandler
type EverythingHandler struct{}

func (h *EverythingHandler) HandleEverything(
	_ miruken.Handles,
	callback interface{},
) miruken.HandleResult {
	switch f := callback.(type) {
	case *Foo:
		f.Inc()
		return miruken.Handled
	case Counter:
		f.Inc()
		f.Inc()
		return miruken.Handled
	default:
		return miruken.NotHandled
	}
}

// SpecificationHandler
type SpecificationHandler struct{}

func (h *SpecificationHandler) HandleFoo(
	_ *struct{ miruken.Handles `bind:"strict"` },
	foo *Foo,
) miruken.HandleResult {
	foo.Inc()
	return miruken.Handled
}

// DependencyHandler
type DependencyHandler struct{}

func (h *DependencyHandler) RequiredDependency(_ miruken.Handles,
	foo *Foo,
	bar *Bar,
) {
	foo.Inc()
}

func (h *DependencyHandler) RequiredSliceDependency(
	_ miruken.Handles,
	boo   *Boo,
	bars []*Bar,
) {
	boo.Inc()
	for _, bar := range bars {
		bar.Inc()
	}
}

func (h *DependencyHandler) OptionalDependency(
	_ miruken.Handles,
	bar *Bar,
	foo *struct{ Value *Foo `bind:"optional"` },
) {
	bar.Inc()
	if foo.Value != nil {
		foo.Value.Inc()
	}
}

func (h *DependencyHandler) OptionalSliceDependency(
	_ miruken.Handles,
	baz  *Baz,
	bars *struct{ Value []*Bar `bind:"optional"` },
) {
	baz.Inc()
	for _, bar := range bars.Value {
		bar.Inc()
	}
}

func (h *DependencyHandler) StrictDependency(
	_ miruken.Handles,
	bam  *Bam,
	bars *struct{ Value []*Bar `bind:"strict"` },
) {
	bam.Inc()
	for _, bar := range bars.Value {
		bar.Inc()
	}
}

type Config struct {
	baseUrl string
	timeout int
}

type Configuration struct {
	config *Config
}

func (c Configuration) Validate(
	typ reflect.Type,
	dep miruken.DependencyArg,
) error {
	argType := dep.ArgType(typ)
	if !reflect.TypeOf(c.config).AssignableTo(argType) {
		return fmt.Errorf("the Configuration resolver expects a %T field", c.config)
	}
	return nil
}

func (c Configuration) Resolve(
	typ         reflect.Type,
	rawCallback interface{},
	dep         miruken.DependencyArg,
	handler     miruken.Handler,
) (reflect.Value, error) {
	if c.config == nil {
		c.config = &Config{
			baseUrl: "https://server/api",
			timeout: 30000,
		}
	}
	return reflect.ValueOf(c.config), nil
}

// DependencyResolverHandler
type DependencyResolverHandler struct{}

func (h *DependencyResolverHandler) UseDependencyResolver(
	_ miruken.Handles,
	foo    *Foo,
	config *struct{ _ Configuration; Value *Config },
) *Config {
	foo.Inc()
	return config.Value
}

// InvalidHandler
type InvalidHandler struct {}

func (h *InvalidHandler) MissingCallback(
	_ miruken.Handles,
) {
}

func (h *InvalidHandler) TooManyReturnValues(
	_ miruken.Handles,
	bar *Bar,
) (int, string, Counter) {
	return 0, "bad", nil
}

func (h *InvalidHandler) SecondReturnMustBeErrorOrHandleResult(
	_ miruken.Handles,
	counter *Counter,
) (Foo, string) {
	return Foo{}, "bad"
}

func (h *InvalidHandler) UntypedInterfaceDependency(
	_ miruken.Handles,
	bar *Bar,
	any  interface{},
) miruken.HandleResult {
	return miruken.Handled
}

type HandlerTestSuite struct {
	suite.Suite
	HandleTypes []reflect.Type
}

func (suite *HandlerTestSuite) SetupTest() {
	suite.HandleTypes = make([]reflect.Type, 0)
	for _, typ := range HandlerTestTypes {
		if !strings.Contains(typ.Elem().Name(), "Invalid") {
			suite.HandleTypes = append(suite.HandleTypes, typ)
		}
	}
}

func (suite *HandlerTestSuite) InferenceRoot() miruken.Handler {
	return miruken.NewRootHandler(miruken.WithHandlerTypes(suite.HandleTypes...))
}

func (suite *HandlerTestSuite) TestHandles() {
	suite.Run("Invariant", func () {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(FooHandler), new(BarHandler)))
		foo     := new(Foo)
		result  := handler.Handle(foo, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
		suite.Equal(1, foo.Count())
	})

	suite.Run("Covariant", func () {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(CounterHandler)))
		foo    := new(Foo)
		result := handler.Handle(foo, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
		suite.Equal(1, foo.Count())
	})

	suite.Run("HandleResult", func () {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(CounterHandler)))
		suite.Run("Handled", func() {
			foo := new(Foo)
			foo.Inc()
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.NotHandled, result)
		})

		suite.Run("NotHandled", func() {
			foo := new(Foo)
			foo.Inc()
			foo.Inc()
			result := handler.Handle(foo, false, nil)
			suite.True(result.IsError())
			suite.Equal("3 is divisible by 3", result.Error().Error())
		})
	})

	suite.Run("Multiple", func () {
		multi   := new(MultiHandler)
		handler := miruken.NewRootHandler(miruken.WithHandlers(multi))
		foo     := new(Foo)

		for i := 0; i < 4; i++ {
			result := handler.Handle(foo, false, nil)
			suite.Equal(miruken.Handled, result)
			suite.Equal(i + 1, foo.Count())
		}

		suite.Equal(4, multi.foo.Count())
		suite.Equal(4, multi.bar.Count())

		result := handler.Handle(foo, false, nil)
		suite.True(result.IsError())
		suite.Equal("count reached 5", result.Error().Error())

		suite.Equal(5, multi.foo.Count())
		suite.Equal(4, multi.bar.Count())
	})

	suite.Run("Everything", func () {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(EverythingHandler)))

		suite.Run("Invariant", func () {
			foo    := new(Foo)
			result := handler.Handle(foo, false, nil)

			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, foo.Count())
		})

		suite.Run("Contravariant", func () {
			bar    := new(Bar)
			result := handler.Handle(bar, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(2, bar.Count())
		})
	})

	suite.Run("Specification", func () {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(SpecificationHandler)))
		suite.Run("Strict", func() {
			foo    := new(Foo)
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, foo.Count())
		})
	})

	suite.Run("Dependencies", func () {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(DependencyHandler)))
		suite.Run("Required", func () {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(miruken.MethodBindingError); ok {
						suite.Equal("RequiredDependency", err.Method.Name)
					} else {
						suite.Fail("Expected MethodBindingError")
					}
				}
			}()
			handler.Handle(new(Foo), false, nil)
		})

		suite.Run("RequiredSlice", func () {
			boo    := new(Boo)
			bars := []interface{}{new(Bar), new(Bar)}
			result := miruken.Build(handler, miruken.With(bars...)).Handle(boo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, boo.Count())
			for _, bar := range bars {
				suite.Equal(1, bar.(*Bar).Count())
			}
		})

		suite.Run("Optional", func () {
			bar    := new(Bar)
			result := handler.Handle(bar, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, bar.Count())
		})

		suite.Run("OptionalWithValue", func () {
			bar    := new(Bar)
			foo    := new(Foo)
			result := miruken.Build(handler, miruken.With(foo)).Handle(bar, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, bar.Count())
			suite.Equal(1, foo.Count())
		})

		suite.Run("OptionalSlice", func () {
			baz    := new(Baz)
			bars   := []interface{}{new(Bar), new(Bar)}
			result := handler.Handle(baz, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, baz.Count())
			result = miruken.Build(handler, miruken.With(bars...)).Handle(baz, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(2, baz.Count())
			for _, bar := range bars {
				suite.Equal(1, bar.(*Bar).Count())
			}
		})

		suite.Run("StrictSlice", func () {
			bam    := new(Bam)
			bars1  := []interface{}{new(Bar), new(Bar)}
			result := miruken.Build(handler, miruken.With(bars1...)).Handle(bam, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.NotHandled, result)
			bars2  := []*Bar{new(Bar), new(Bar)}
			result  = miruken.Build(handler, miruken.With(bars2)).Handle(bam, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, bam.Count())
			for _, bar := range bars2 {
				suite.Equal(1, bar.Count())
			}
		})

		suite.Run("CustomResolver", func() {
			handler := miruken.NewRootHandler(miruken.WithHandlers(new(DependencyResolverHandler)))
			var config *Config
			if err := miruken.Invoke(handler, new(Foo), &config); err == nil {
				suite.NotNil(*config)
				suite.Equal("https://server/api", config.baseUrl)
				suite.Equal(30000, config.timeout)
			} else {
				suite.Failf("unexpected error", err.Error())
			}
		})
	})

	suite.Run("CallSemantics", func () {
		suite.Run("BestEffort", func () {
			handler := miruken.NewRootHandler(miruken.WithHandlers(new(BarHandler)), miruken.WithBestEffort)
			foo     := new(Foo)
			result  := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(0, foo.Count())
		})

		suite.Run("Broadcast", func () {
			handler := miruken.NewRootHandler(miruken.WithHandlers(
				new(FooHandler), new(FooHandler), new(BarHandler)))
			foo     := new(Foo)
			result  := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, foo.Count())

			result = miruken.Build(handler, miruken.WithBroadcast).Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(3, foo.Count())
		})
	})

	suite.Run("Command", func () {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(CounterHandler)))
		suite.Run("Invoke", func () {
			suite.Run("Invariant", func() {
				var foo *Foo
				if err := miruken.Invoke(handler, new(Foo), &foo); err == nil {
					suite.NotNil(foo)
					suite.Equal(1, foo.Count())
				} else {
					suite.Failf("unexpected error", err.Error())
				}
			})

			suite.Run("Contravariant", func() {
				var foo interface{}
				if err := miruken.Invoke(handler, new(Foo), &foo); err == nil {
					suite.NotNil(foo)
					suite.IsType(&Foo{}, foo)
					suite.Equal(1, foo.(*Foo).Count())
				} else {
					suite.Failf("unexpected error", err.Error())
				}
			})

			suite.Run("BestEffort", func () {
				handler := miruken.NewRootHandler(miruken.WithHandlers(new(BarHandler)), miruken.WithBestEffort)
				var foo *Foo
				if err := miruken.Invoke(handler, new(Foo), &foo); err == nil {
					suite.Nil(foo)
				} else {
					suite.Failf("unexpected error", err.Error())
				}
			})
		})

		suite.Run("InvokeAll", func () {
			handler := miruken.NewRootHandler(miruken.WithHandlers(
				new(CounterHandler), new(SpecificationHandler)))
			suite.Run("Invariant", func () {
				var foo []*Foo
				if err := miruken.InvokeAll(handler, new(Foo), &foo); err == nil {
					suite.NotNil(foo)
					suite.Len(foo, 1)
					suite.Equal(2, foo[0].Count())
				} else {
					suite.Failf("unexpected error", err.Error())
				}
			})
		})
	})

	suite.Run("Invalid", func () {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(*miruken.HandlerDescriptorError); ok {
					failures := 0
					var errMethod miruken.MethodBindingError
					for reason := errors.Unwrap(err.Reason);
						errors.As(reason, &errMethod); reason = errors.Unwrap(reason) {
						failures++
					}
					suite.Equal(4, failures)
				} else {
					suite.Fail("Expected HandlerDescriptorError")
				}
			}
		}()
		miruken.NewRootHandler(miruken.WithHandlers(new(InvalidHandler)))
	})
}

// FooProvider
type FooProvider struct {
	foo Foo
}

func (f *FooProvider) ProvideFoo(_ miruken.Provides) *Foo {
	f.foo.Inc()
	return &f.foo
}

// ListProvider
type ListProvider struct {}

func (f *ListProvider) ProvideFooSlice(_ miruken.Provides) []*Foo {
	return []*Foo{{Counted{1}}, {Counted{2}}}
}

func (f *ListProvider) ProvideFooArray(_ miruken.Provides) [2]*Bar {
	return [2]*Bar{{Counted{3}}, {Counted{4}}}
}

// MultiProvider
type MultiProvider struct {
	foo Foo
	bar Bar
}

func (p *MultiProvider) Constructor(
	_ *struct{ miruken.Creates },
) {
	p.foo.Inc()
}

func (p *MultiProvider) ProvideFoo(_ miruken.Provides) *Foo {
	p.foo.Inc()
	return &p.foo
}

func (p *MultiProvider) ProvideBar(_ miruken.Provides) (*Bar, miruken.HandleResult) {
	if p.bar.Inc() % 3 == 0 {
		return &p.bar, miruken.NotHandled.WithError(
			fmt.Errorf("%v is divisible by 3", p.bar.Count()))
	}
	if p.bar.Inc() % 2 == 0 {
		return &p.bar, miruken.NotHandled
	}
	return &p.bar, miruken.Handled
}

// SpecificationProvider
type SpecificationProvider struct{
	foo Foo
	bar Bar
}

func (p *SpecificationProvider) Constructor(baz Baz) {
	p.foo.count = baz.Count()
}

func (p *SpecificationProvider) ProvidesFoo(
	_ *struct{
		miruken.Provides;
		miruken.Creates
	  },
) *Foo {
	p.foo.Inc()
	return &p.foo
}

func (p *SpecificationProvider) ProvidesBar(
	_ *struct{ miruken.Provides `bind:"strict"` },
) []*Bar {
	p.bar.Inc()
	return []*Bar{&p.bar, {}}
}

type GenericProvider struct{}

func (p *GenericProvider) Provide(
	_ miruken.Provides,
	inquiry *miruken.Inquiry,
) interface{} {
	if inquiry.Key() == reflect.TypeOf((*Foo)(nil)) {
		return &Foo{}
	}
	if inquiry.Key() == reflect.TypeOf((*Bar)(nil)) {
		return &Bar{}
	}
	return nil
}

// InvalidProvider
type InvalidProvider struct {}

func (p *InvalidProvider) MissingReturnValue(_ miruken.Provides) {
}

func (p *InvalidProvider) TooManyReturnValues(
	_ miruken.Provides,
) (*Foo, string, Counter) {
	return nil, "bad", nil
}

func (p *InvalidProvider) InvalidHandleResultReturnValue(
	_ miruken.Provides,
) miruken.HandleResult {
	return miruken.Handled
}

func (p *InvalidProvider) InvalidErrorReturnValue(
	_ miruken.Provides,
) error {
	return errors.New("not good")
}

func (p *InvalidProvider) SecondReturnMustBeErrorOrHandleResult(
	_ miruken.Provides,
) (*Foo, string) {
	return &Foo{}, "bad"
}

func (p *InvalidProvider) UntypedInterfaceDependency(
	_ miruken.Provides,
	any interface{},
) *Foo {
	return &Foo{}
}

func (suite *HandlerTestSuite) TestProvides() {
	suite.Run("Implied", func () {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(FooProvider)))
		var fooProvider *FooProvider
		err := miruken.Resolve(handler, &fooProvider)
		suite.Nil(err)
		suite.NotNil(fooProvider)
	})

	suite.Run("Invariant", func () {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(FooProvider)))
		var foo *Foo
		err := miruken.Resolve(handler, &foo)
		suite.Nil(err)
		suite.Equal(1, foo.Count())
	})

	suite.Run("Covariant", func () {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(FooProvider)))
		var counter Counter
		err := miruken.Resolve(handler, &counter)
		suite.Nil(err)
		suite.Equal(1, counter.Count())
		if foo, ok := counter.(*Foo); !ok {
			suite.Fail(fmt.Sprintf("expected *Foo, but found %T", foo))
		}
	})

	suite.Run("NotHandledReturnNil", func () {
		handler := miruken.NewRootHandler()
		var foo *Foo
		err := miruken.Resolve(handler, &foo)
		suite.Nil(err)
		suite.Nil(foo)
	})

	suite.Run("Generic", func () {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(GenericProvider)))
		var foo *Foo
		err := miruken.Resolve(handler, &foo)
		suite.Nil(err)
		suite.Equal(0, foo.Count())
		var bar *Bar
		err = miruken.Resolve(handler, &bar)
		suite.Nil(err)
		suite.Equal(0, bar.Count())
	})

	suite.Run("Multiple", func () {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(MultiProvider)))
		var foo *Foo
		err := miruken.Resolve(handler, &foo)
		suite.Nil(err)
		suite.Equal(1, foo.Count())

		var bar *Bar
		err = miruken.Resolve(handler, &bar)
		suite.Nil(err)
		suite.Nil(bar)

		err = miruken.Resolve(handler, &bar)
		suite.NotNil(err)
		suite.Equal("3 is divisible by 3", err.Error())
		suite.Nil(bar)
	})

	suite.Run("Specification", func () {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(SpecificationProvider)))

		suite.Run("Invariant", func () {
			var foo *Foo
			err := miruken.Resolve(handler, &foo)
			suite.Nil(err)
			suite.Equal(1, foo.Count())
		})

		suite.Run("Strict", func () {
			var bar *Bar
			err := miruken.Resolve(handler, &bar)
			suite.Nil(err)
			suite.Nil(bar)

			var bars []*Bar
			err = miruken.Resolve(handler, &bars)
			suite.Nil(err)
			suite.NotNil(bars)
			suite.Equal(2, len(bars))
		})
	})

	suite.Run("Lists", func () {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(ListProvider)))

		suite.Run("Slice", func () {
			var foo *Foo
			err := miruken.Resolve(handler, &foo)
			suite.Nil(err)
			suite.NotNil(foo)
		})

		suite.Run("Array", func () {
			var bar *Bar
			err := miruken.Resolve(handler, &bar)
			suite.Nil(err)
			suite.NotNil(bar)
		})
	})

	suite.Run("Constructor", func () {
		handler := suite.InferenceRoot()

		suite.Run("NoInit", func () {
			var fooProvider *FooProvider
			err := miruken.Resolve(handler, &fooProvider)
			suite.NotNil(fooProvider)
			suite.Nil(err)
		})

		suite.Run("Constructor", func () {
			var multiProvider *MultiProvider
			err := miruken.Resolve(handler, &multiProvider)
			suite.NotNil(multiProvider)
			suite.Equal(1, multiProvider.foo.Count())
			suite.Nil(err)
		})

		suite.Run("ConstructorDependencies", func () {
			handler := miruken.NewRootHandler(miruken.WithHandlerTypes(reflect.TypeOf((*SpecificationProvider)(nil))))
			var specProvider *SpecificationProvider
			err := miruken.Resolve(miruken.Build(handler, miruken.With(Baz{Counted{2}})), &specProvider)
			suite.NotNil(specProvider)
			suite.Equal(2, specProvider.foo.Count())
			suite.Equal(0, specProvider.bar.Count())
			suite.Nil(err)
		})
	})

	suite.Run("Infer", func () {
		handler := suite.InferenceRoot()

		suite.Run("Invariant", func() {
			foo := new(Foo)
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, foo.Count())
		})
	})

	suite.Run("ResolveAll", func () {
		suite.Run("Invariant", func () {
			handler := miruken.NewRootHandler(miruken.WithHandlers(
				new(FooProvider), new(MultiProvider), new (SpecificationProvider)))

			var foo []*Foo
			if err := miruken.ResolveAll(handler, &foo); err == nil {
				suite.NotNil(foo)
				suite.Len(foo, 3)
				suite.True(foo[0] != foo[1])
			} else {
				suite.Failf("unexpected error", err.Error())
			}
		})

		suite.Run("Covariant", func () {
			handler := miruken.NewRootHandler(miruken.WithHandlers(new(ListProvider)))
			var counted []Counter
			if err := miruken.ResolveAll(handler, &counted); err == nil {
				suite.NotNil(counted)
				suite.Len(counted, 4)
			} else {
				suite.Failf("unexpected error", err.Error())
			}
		})

		suite.Run("Empty", func () {
			handler := miruken.NewRootHandler(miruken.WithHandlers(new(FooProvider)))
			var bars []*Bar
			err := miruken.ResolveAll(handler, &bars)
			suite.Nil(err)
			suite.NotNil(bars)
		})
	})

	suite.Run("With", func () {
		handler := miruken.NewRootHandler()
		var fooProvider *FooProvider
		err := miruken.Resolve(handler, &fooProvider)
		suite.Nil(err)
		suite.Nil(fooProvider)
		err = miruken.Resolve(miruken.Build(handler, miruken.With(new(FooProvider))), &fooProvider)
		suite.Nil(err)
		suite.NotNil(fooProvider)
	})

	suite.Run("Invalid", func () {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(*miruken.HandlerDescriptorError); ok {
					failures := 0
					var errMethod miruken.MethodBindingError
					for reason := errors.Unwrap(err.Reason);
						errors.As(reason, &errMethod); reason = errors.Unwrap(reason) {
						failures++
					}
					suite.Equal(6, failures)
				} else {
					suite.Fail("Expected HandlerDescriptorError")
				}
			}
		}()

		miruken.NewRootHandler(miruken.WithHandlers(new(InvalidProvider)))
	})
}

func (suite *HandlerTestSuite) TestCreates() {
	suite.Run("Invariant", func() {
		handler := miruken.NewRootHandler(miruken.WithHandlers(new(SpecificationProvider)))
		var foo *Foo
		err := miruken.Create(handler, &foo)
		suite.Nil(err)
		suite.Equal(1, foo.Count())
	})

	suite.Run("Infer", func() {
		handler := suite.InferenceRoot()
		var multiProvider *MultiProvider
		err := miruken.Create(handler, &multiProvider)
		suite.NotNil(multiProvider)
		suite.Nil(err)
	})
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}