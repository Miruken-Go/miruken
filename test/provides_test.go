package test

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
	"time"
)

// FooProvider
type FooProvider struct {
	foo Foo
}

func (f *FooProvider) ProvideFoo(*miruken.Provides) *Foo {
	f.foo.Inc()
	return &f.foo
}

// ListProvider
type ListProvider struct{}

func (f *ListProvider) ProvideFooSlice(*miruken.Provides) []*Foo {
	return []*Foo{{Counted{1}}, {Counted{2}}}
}

func (f *ListProvider) ProvideFooArray(*miruken.Provides) [2]*Bar {
	return [2]*Bar{{Counted{3}}, {Counted{4}}}
}

// MultiProvider
type MultiProvider struct {
	foo Foo
	bar Bar
}

func (p *MultiProvider) Constructor(*miruken.Creates) {
	p.foo.Inc()
}

func (p *MultiProvider) ProvideFoo(*miruken.Provides) *Foo {
	p.foo.Inc()
	return &p.foo
}

func (p *MultiProvider) ProvideBar(*miruken.Provides) (*Bar, miruken.HandleResult) {
	count := p.bar.Inc()
	if count % 3 == 0 {
		return &p.bar, miruken.NotHandled.WithError(
			fmt.Errorf("%v is divisible by 3", p.bar.Count()))
	}
	if count % 2 == 0 {
		return &p.bar, miruken.NotHandled
	}
	return &p.bar, miruken.Handled
}

// SpecificationProvider
type SpecificationProvider struct {
	foo Foo
	bar Bar
}

func (p *SpecificationProvider) Constructor(baz Baz) {
	p.foo.count = baz.Count()
}

func (p *SpecificationProvider) ProvideFoo(
	_*struct{
		miruken.Provides
		miruken.Creates
	  },
) *Foo {
	p.foo.Inc()
	return &p.foo
}

func (p *SpecificationProvider) ProvideBar(
	_*struct{ miruken.Provides; miruken.Strict },
) []*Bar {
	p.bar.Inc()
	return []*Bar{&p.bar, {}}
}

// KeyProvider
type KeyProvider struct {
	foo Foo
}

func (p *KeyProvider) ProvideKey(
	_*struct{
		miruken.Provides `key:"Foo"`
	  },
) *Foo {
	p.foo.Inc()
	return &p.foo
}

// OpenProvider
type OpenProvider struct{}

func (p *OpenProvider) Provide(
	provides *miruken.Provides,
) any {
	if key := provides.Key(); key == miruken.TypeOf[*Foo]() {
		return &Foo{}
	} else if key == miruken.TypeOf[*Bar]() {
		return &Bar{}
	} else if key == "Foo" {
		return &Foo{Counted{1}}
	}
	return nil
}

// UnmanagedHandler
type UnmanagedHandler struct {}

func (u *UnmanagedHandler) NoConstructor() {}

// SimpleAsyncProvider
type SimpleAsyncProvider struct {
	foo Foo
}

func (p *SimpleAsyncProvider) ProvideFoo(
	*miruken.Provides,
) *promise.Promise[*Foo] {
	p.foo.Inc()
	return promise.Then(promise.Delay(5 * time.Millisecond),
		func(void miruken.Void) *Foo {
			return &p.foo
		})
}

// ComplexAsyncProvider
type ComplexAsyncProvider struct {
	bar Bar
}

func (p *ComplexAsyncProvider) Constructor(
	_*struct{ miruken.Provides; miruken.Creates },
) *promise.Promise[miruken.Void] {
	return promise.Then(
		promise.Delay(2 * time.Millisecond),
			func(miruken.Void) miruken.Void {
				p.bar.Inc()
				return miruken.Void{}
			})
}

func (p *ComplexAsyncProvider) ProvideBar(
	_*miruken.Provides,
	foo *Foo,
) *Bar {
	p.bar.Inc()
	foo.Inc()
	return &p.bar
}

// InvalidProvider
type InvalidProvider struct {}

func (p *InvalidProvider) MissingReturnValue(*miruken.Provides) {
}

func (p *InvalidProvider) TooManyReturnValues(
	*miruken.Provides,
) (*Foo, string, Counter) {
	return nil, "bad", nil
}

func (p *InvalidProvider) InvalidHandleResultReturnValue(
	*miruken.Provides,
) miruken.HandleResult {
	return miruken.Handled
}

func (p *InvalidProvider) InvalidErrorReturnValue(
	*miruken.Provides,
) error {
	return errors.New("not good")
}

func (p *InvalidProvider) SecondReturnMustBeErrorOrHandleResult(
	*miruken.Provides,
) (*Foo, string) {
	return &Foo{}, "bad"
}

func (p *InvalidProvider) UntypedInterfaceDependency(
	_*miruken.Provides,
	any any,
) *Foo {
	return &Foo{}
}

func ProvideBar(*miruken.Provides) (*Bar, miruken.HandleResult) {
	bar := &Bar{}
	bar.Inc()
	bar.Inc()
	return bar, miruken.Handled
}

type ProvidesTestSuite struct {
	suite.Suite
}

func (suite *ProvidesTestSuite) Setup() (miruken.Handler, error) {
	return miruken.Setup(TestFeature, miruken.ExcludeHandlerSpecs(
		func (spec miruken.HandlerSpec) bool {
			switch ts := spec.(type) {
			case miruken.HandlerTypeSpec:
				return strings.Contains(ts.Name(), "Invalid")
			default:
				return false
			}
		}))
}

func (suite *ProvidesTestSuite) SetupWith(
	features ... miruken.Feature,
) (miruken.Handler, error) {
	return miruken.Setup(features...)
}

func (suite *ProvidesTestSuite) TestProvides() {
	suite.Run("Implied", func () {
		handler, _ := suite.SetupWith(miruken.Handlers(new(FooProvider)))
		fooProvider, _, err := miruken.Resolve[*FooProvider](handler)
		suite.Nil(err)
		suite.NotNil(fooProvider)
	})

	suite.Run("Invariant", func () {
		handler, _ := suite.SetupWith(
			miruken.HandlerSpecs(&FooProvider{}),
			miruken.Handlers(new(FooProvider)))
		foo, _, err := miruken.Resolve[*Foo](handler)
		suite.Nil(err)
		suite.Equal(1, foo.Count())
	})

	suite.Run("Covariant", func () {
		handler, _ := suite.SetupWith(
			miruken.HandlerSpecs(&FooProvider{}),
			miruken.Handlers(new(FooProvider)))
		counter, _, err := miruken.Resolve[Counter](handler)
		suite.Nil(err)
		suite.Equal(1, counter.Count())
		if foo, ok := counter.(*Foo); !ok {
			suite.Fail(fmt.Sprintf("expected *Foo, but found %T", foo))
		}
	})

	suite.Run("NotHandledReturnNil", func () {
		handler, _ := suite.SetupWith()
		foo, _, err := miruken.Resolve[*Foo](handler)
		suite.Nil(err)
		suite.Nil(foo)
	})

	suite.Run("Key", func () {
		handler, _ := suite.SetupWith(
			miruken.HandlerSpecs(&KeyProvider{}),
			miruken.Handlers(new(FooProvider)))
		foo, _, err := miruken.ResolveKey[*Foo](handler, "Foo")
		suite.Nil(err)
		suite.Equal(1, foo.Count())
	})

	suite.Run("Open", func () {
		handler, _ := suite.SetupWith(
			miruken.HandlerSpecs(&OpenProvider{}),
			miruken.Handlers(new(OpenProvider)))
		foo, _, err := miruken.Resolve[*Foo](handler)
		suite.Nil(err)
		suite.Equal(0, foo.Count())
		bar, _, err := miruken.Resolve[*Bar](handler)
		suite.Nil(err)
		suite.Equal(0, bar.Count())
		foo, _, err = miruken.ResolveKey[*Foo](handler, "Foo")
		suite.Nil(err)
		suite.Equal(1, foo.Count())
	})

	suite.Run("Multiple", func () {
		handler, _ := suite.SetupWith(
			miruken.HandlerSpecs(&MultiProvider{}),
			miruken.Handlers(new(MultiProvider)))
		foo, _, err := miruken.Resolve[*Foo](handler)
		suite.Nil(err)
		suite.Equal(1, foo.Count())

		bar, _, err := miruken.Resolve[*Bar](handler)
		suite.Nil(err)
		suite.Equal(1, bar.Count())

		bar, _, err = miruken.Resolve[*Bar](handler)
		suite.NotNil(err)
		suite.Nil(bar)
	})

	suite.Run("Specification", func () {
		handler, _ := suite.SetupWith(
			miruken.HandlerSpecs(&SpecificationProvider{}),
			miruken.Handlers(new(SpecificationProvider)))

		suite.Run("Invariant", func () {
			foo, _, err := miruken.Resolve[*Foo](handler)
			suite.Nil(err)
			suite.Equal(1, foo.Count())
		})

		suite.Run("Strict", func () {
			bar, _, err := miruken.Resolve[*Bar](handler)
			suite.Nil(err)
			suite.Nil(bar)

			bars, _, err := miruken.Resolve[[]*Bar](handler)
			suite.Nil(err)
			suite.NotNil(bars)
			suite.Equal(2, len(bars))
		})
	})

	suite.Run("Lists", func () {
		handler, _ := suite.SetupWith(
			miruken.HandlerSpecs(&ListProvider{}),
			miruken.Handlers(new(ListProvider)))

		suite.Run("Slice", func () {
			foo, _, err := miruken.Resolve[*Foo](handler)
			suite.Nil(err)
			suite.NotNil(foo)
		})

		suite.Run("Array", func () {
			bar, _, err := miruken.Resolve[*Bar](handler)
			suite.Nil(err)
			suite.NotNil(bar)
		})
	})

	suite.Run("Constructor", func () {
		handler, _ := suite.Setup()

		suite.Run("NoInit", func () {
			fooProvider, _, err := miruken.Resolve[*FooProvider](handler)
			suite.NotNil(fooProvider)
			suite.Nil(err)
		})

		suite.Run("Constructor", func () {
			multiProvider, _, err := miruken.Resolve[*MultiProvider](handler)
			suite.NotNil(multiProvider)
			suite.Equal(1, multiProvider.foo.Count())
			suite.Nil(err)
		})

		suite.Run("ConstructorDependencies", func () {
			handler, _ := suite.SetupWith(
				miruken.HandlerSpecs(&SpecificationProvider{}))
			specProvider, _, err := miruken.Resolve[*SpecificationProvider](
				miruken.BuildUp(handler, miruken.With(Baz{Counted{2}})))
			suite.NotNil(specProvider)
			suite.Equal(2, specProvider.foo.Count())
			suite.Equal(0, specProvider.bar.Count())
			suite.Nil(err)
		})

		suite.Run("NoConstructor", func () {
			unmanaged, _, err := miruken.Resolve[*UnmanagedHandler](handler)
			suite.Nil(err)
			suite.Nil(unmanaged)
		})
	})

	suite.Run("Infer", func () {
		suite.Run("Invariant", func() {
			handler, _ := suite.SetupWith(
				miruken.HandlerSpecs(
					&SpecificationHandler{}))
			foo := new(Foo)
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.Handled, result)
			suite.Equal(1, foo.Count())
		})

		suite.Run("Open", func () {
			handler, _ := suite.Setup()
			foo, fp, err := miruken.ResolveAll[*Foo](handler)
			suite.Nil(err)
			if fp != nil {
				foo, err = fp.Await()
				suite.Nil(err)
			}
			// 1 from FooProvider.ProvideFoo
			// 2 from ListProvider.ProvideFooSlice
			// 1 from MultiProvider.ProvideFoo
			// 1 from OpenProvider.Provides
			// 1 from SimpleAsyncProvider.ProvideFoo
			// None from SpecificationProvider.ProvideFoo since it
			//   depends on an unsatisfied Baz
			// 5 total
			suite.Equal(6, len(foo))
		})

		suite.Run("Disable", func() {
			handler, _ := suite.SetupWith(
				miruken.Handlers(new(FooProvider)),
				miruken.NoInference)
			foo := new(Foo)
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.NotHandled, result)
		})
	})

	suite.Run("ResolveAll", func () {
		suite.Run("Invariant", func () {
			handler, _ := suite.SetupWith(
				miruken.HandlerSpecs(
					&FooProvider{},
					&MultiProvider{},
					&SpecificationProvider{}),
				miruken.Handlers(
					new(FooProvider), new(MultiProvider), new (SpecificationProvider)))

			if foo, _, err := miruken.ResolveAll[*Foo](handler); err == nil {
				suite.NotNil(foo)
				// 3 from each of the 3 explicit instances (1)
				// 2 for inference of *FooProvider (1) which includes explicit instance (1)
				// 2 for inference of *MultiProvider (1) which includes explicit instance (1)
				// 1 for inference of *SpecificationProvider (1) which excludes constructed
				//   instance since it has an unsatisfied dependency on Baz
				// 8 total
				suite.Len(foo, 8)
				suite.True(foo[0] != foo[1])
			} else {
				suite.Fail("unexpected error", err.Error())
			}
		})

		suite.Run("Covariant", func () {
			handler, _ := suite.SetupWith(
				miruken.HandlerSpecs(&ListProvider{}),
				miruken.Handlers(new(ListProvider)))
			if counted, _, err := miruken.ResolveAll[Counter](handler); err == nil {
				suite.NotNil(counted)
				// 4 from 2 methods on explicit *ListProvider
				// 8 for inference of *ListProvider (4) which includes explicit instance (4)
				// 12 total
				suite.Len(counted, 12)
			} else {
				suite.Fail("unexpected error", err.Error())
			}
		})

		suite.Run("Empty", func () {
			handler, _ := suite.SetupWith(miruken.Handlers(new(FooProvider)))
			bars, _, err := miruken.ResolveAll[*Bar](handler)
			suite.Nil(err)
			suite.Nil(bars)
		})
	})

	suite.Run("With", func () {
		handler, _ := miruken.Setup()
		fooProvider, _, err := miruken.Resolve[*FooProvider](handler)
		suite.Nil(err)
		suite.Nil(fooProvider)
		fooProvider, _, err = miruken.Resolve[*FooProvider](
			miruken.BuildUp(handler, miruken.With(new(FooProvider))))
		suite.Nil(err)
		suite.NotNil(fooProvider)
	})

	suite.Run("Invalid", func () {
		failures := 0
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(*miruken.HandlerDescriptorError); ok {
					var errMethod *miruken.MethodBindingError
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
		_, err := suite.SetupWith(
			miruken.HandlerSpecs(&InvalidProvider{}),
			miruken.Handlers(new(InvalidProvider)))
		suite.Nil(err)
		suite.Fail("should cause panic")
	})

	suite.Run("Function Binding", func () {
		suite.Run("Implied", func() {
			handler, _ := suite.SetupWith(miruken.HandlerSpecs(ProvideBar))
			bar, _, err := miruken.Resolve[*Bar](handler)
			suite.Nil(err)
			suite.NotNil(bar)
			suite.Equal(2, bar.Count())
		})
	})
}

func (suite *ProvidesTestSuite) TestProvidesAsync() {
	suite.Run("Simple", func () {
		suite.Run("Returns Promise", func() {
			handler, _ := suite.SetupWith(
				miruken.HandlerSpecs(&SimpleAsyncProvider{}))
			foo, pf, err := miruken.Resolve[*Foo](handler)
			suite.Nil(err)
			suite.Nil(foo)
			suite.NotNil(pf)
			foo, err = pf.Await()
			suite.Nil(err)
			suite.Equal(1, foo.Count())
		})
	})

	suite.Run("Complex", func () {
		suite.Run("Returns Promise", func() {
			handler, _ := suite.SetupWith(
				miruken.HandlerSpecs(&SimpleAsyncProvider{}),
				miruken.HandlerSpecs(&ComplexAsyncProvider{}))
			bar, pb, err := miruken.Resolve[*Bar](handler)
			suite.Nil(err)
			suite.Nil(bar)
			suite.NotNil(pb)
			bar, err = pb.Await()
			suite.Nil(err)
			suite.Equal(2, bar.Count())
			foo, pf, err := miruken.Resolve[*Foo](handler)
			suite.Nil(err)
			suite.Nil(foo)
			suite.NotNil(pf)
			foo, err = pf.Await()
			suite.Nil(err)
			suite.Equal(3, foo.Count())
		})
	})
}

func TestProvidesTestSuite(t *testing.T) {
	suite.Run(t, new(ProvidesTestSuite))
}
