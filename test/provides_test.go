package test

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
	"time"
)

// FooProvider
type FooProvider struct {
	foo Foo
}

func (f *FooProvider) ProvideFoo(*provides.It) *Foo {
	f.foo.Inc()
	return &f.foo
}

// ListProvider
type ListProvider struct{}

func (f *ListProvider) ProvideFooSlice(*provides.It) []*Foo {
	return []*Foo{{Counted{1}}, {Counted{2}}}
}

func (f *ListProvider) ProvideFooArray(*provides.It) [2]*Bar {
	return [2]*Bar{{Counted{3}}, {Counted{4}}}
}

// MultiProvider
type MultiProvider struct {
	foo Foo
	bar Bar
}

func (p *MultiProvider) Constructor(
	_*struct{
	    c creates.It
		p provides.It
		provides.Single
	  },
) {
	p.foo.Inc()
}

func (p *MultiProvider) ProvideFoo(*provides.It) *Foo {
	p.foo.Inc()
	return &p.foo
}

func (p *MultiProvider) ProvideBar(*provides.It) (*Bar, miruken.HandleResult) {
	count := p.bar.Inc()
	if count % 3 == 0 {
		return nil, miruken.NotHandled.WithError(
			fmt.Errorf("%v is divisible by 3", p.bar.Count()))
	}
	if count % 2 == 0 {
		return nil, miruken.NotHandled
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
		p provides.It
		c creates.It
	  },
) *Foo {
	p.foo.Inc()
	return &p.foo
}

func (p *SpecificationProvider) ProvideBar(
	_*struct{
		provides.It; provides.Strict
	  },
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
		provides.It `key:"Foo"`
	  },
) *Foo {
 	p.foo.Inc()
	return &p.foo
}

// KeyConsumer
type KeyConsumer struct {
	foo *Foo
}

func (c *KeyConsumer) Constructor(
	_ *provides.It,
	_*struct{args.Key `of:"Foo"`}, foo *Foo,
) {
	foo.Inc()
	c.foo = foo
}

func (c *KeyConsumer) HandleBar(
	_ *handles.It, _ Bar,
	_*struct{args.Key `of:"Foo"`}, foo *Foo,
) *Foo {
	foo.Inc()
	return foo
}

// OpenProvider
type OpenProvider struct {
	foo Foo
	bar Boo
	baz Baz
	bam Bam
}

func (p *OpenProvider) ProvideSingletons(
	_*struct{provides.Single}, it *provides.It,
) any {
	if key := it.Key(); key == internal.TypeOf[*Foo]() {
		p.foo.Inc()
		return &p.foo
	} else if key == internal.TypeOf[*Bar]() {
		p.bar.Inc()
		p.bar.Inc()
		return &p.bar
	} else if key == "Foo" {
		p.foo.Inc()
		return &p.foo
	}
	return nil
}

func (p *OpenProvider) ProvideScoped(
	_*struct{context.Scoped}, it *provides.It,
) any {
	if key := it.Key(); key == internal.TypeOf[*Baz]() {
		p.baz.Inc()
		return &p.baz
	} else if key == internal.TypeOf[*Bam]() {
		p.bam.Inc()
		p.bam.Inc()
		return &p.bam
	} else if key == "Baz" {
		p.baz.Inc()
		return &p.baz
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
	*provides.It,
) *promise.Promise[*Foo] {
	p.foo.Inc()
	return promise.Then(promise.Delay(5 * time.Millisecond),
		func(any) *Foo { return &p.foo })
}

// ComplexAsyncProvider
type ComplexAsyncProvider struct {
	bar Bar
}

func (p *ComplexAsyncProvider) Constructor(
	_*struct{
		p provides.It
		c creates.It
	  },
) *promise.Promise[any] {
	return promise.Then(
		promise.Delay(2 * time.Millisecond),
			func(any) any {
				p.bar.Inc()
				return nil
			})
}

func (p *ComplexAsyncProvider) Init() {
	p.bar.Inc()
}

func (p *ComplexAsyncProvider) ProvideBar(
	_ *provides.It,
	foo *Foo,
) *Bar {
	p.bar.Inc()
	foo.Inc()
	return &p.bar
}

// AsyncArgProvider
type AsyncArgProvider struct {
	cp  *ComplexAsyncProvider
	foo *Foo
}

func (p *AsyncArgProvider) Constructor(
	_ *provides.It,
	cp *ComplexAsyncProvider,
	foo *Foo,
) *promise.Promise[any] {
	p.cp =  cp
	p.foo = foo
	return promise.Delay(1 * time.Millisecond)
}

func (p *AsyncArgProvider) Init() {
	p.foo.Inc()
}

func (p *AsyncArgProvider) InitAsync(
) *promise.Promise[any] {
	p.foo.Inc()
	return promise.Delay(1 * time.Millisecond)
}

func (p *AsyncArgProvider) ExplicitInit(
	_ provides.Init,
) *promise.Promise[any] {
	p.foo.Inc()
	return promise.Delay(1 * time.Millisecond)
}

func (p *AsyncArgProvider) ProvideBar(
	pr *provides.It,
) *Bar {
	p.foo.Inc()
	return p.cp.ProvideBar(pr, p.foo)
}

// InvalidProvider
type InvalidProvider struct {}

func (p *InvalidProvider) MissingReturnValue(*provides.It) {
}

func (p *InvalidProvider) TooManyReturnValues(
	*provides.It,
) (*Foo, string, Counter) {
	return nil, "bad", nil
}

func (p *InvalidProvider) InvalidHandleResultReturnValue(
	*provides.It,
) miruken.HandleResult {
	return miruken.Handled
}

func (p *InvalidProvider) InvalidErrorReturnValue(
	*provides.It,
) error {
	return errors.New("not good")
}

func (p *InvalidProvider) SecondReturnMustBeErrorOrHandleResult(
	*provides.It,
) (*Foo, string) {
	return &Foo{}, "bad"
}

func (p *InvalidProvider) UntypedInterfaceDependency(
	_ *provides.It,
	any any,
) *Foo {
	return &Foo{}
}

func ProvideBar(*provides.It) (*Bar, miruken.HandleResult) {
	bar := &Bar{}
	bar.Inc()
	bar.Inc()
	return bar, miruken.Handled
}

type ProvidesTestSuite struct {
	suite.Suite
}

func (suite *ProvidesTestSuite) Setup() (miruken.Handler, error) {
	return setup.New(TestFeature).ExcludeSpecs(
		func (spec miruken.HandlerSpec) bool {
			switch ts := spec.(type) {
			case miruken.TypeSpec:
				return strings.Contains(ts.Name(), "Invalid")
			default:
				return false
			}
		}).Handler()
}

func (suite *ProvidesTestSuite) TestProvides() {
	suite.Run("Implied", func () {
		handler, _ := setup.New().Handlers(new(FooProvider)).Handler()
		fooProvider, _, ok, err := miruken.Resolve[*FooProvider](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.NotNil(fooProvider)
	})

	suite.Run("Invariant", func () {
		handler, _ := setup.New().Specs(&FooProvider{}).Handler()
		foo, _, ok, err := miruken.Resolve[*Foo](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(1, foo.Count())
	})

	suite.Run("Covariant", func () {
		handler, _ := setup.New().Specs(&FooProvider{}).Handler()
		counter, _, ok, err := miruken.Resolve[Counter](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(1, counter.Count())
		if foo, ok := counter.(*Foo); !ok {
			suite.Fail(fmt.Sprintf("expected *Foo, but found %T", foo))
		}

		counter, _, ok, err = miruken.Resolve[Counter](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(2, counter.Count())
		if foo, ok := counter.(*Foo); !ok {
			suite.Fail(fmt.Sprintf("expected *Foo, but found %T", foo))
		}
	})

	suite.Run("NotHandledReturnNil", func () {
		handler, _ := setup.New().Handler()
		foo, _, ok, err := miruken.Resolve[*Foo](handler)
		suite.False(ok)
		suite.Nil(err)
		suite.Nil(foo)
	})

	suite.Run("Key", func () {
		suite.Run("Provide", func() {
			handler, _ := setup.New().Specs(&KeyProvider{}).Handler()
			foo, _, ok, err := miruken.ResolveKey[*Foo](handler, "Foo")
			suite.True(ok)
			suite.Nil(err)
			suite.Equal(1, foo.Count())
		})

		suite.Run("Consume", func() {
			suite.Run("Constructor", func() {
				handler, _ := setup.New().Specs(&KeyProvider{}, &KeyConsumer{}).Handler()
				kc, _, ok, err := miruken.Resolve[*KeyConsumer](handler)
				suite.True(ok)
				suite.Nil(err)
				suite.NotNil(kc)
				suite.Equal(2, kc.foo.Count())
			})

			suite.Run("Parameter", func() {
				handler, _ := setup.New().Specs(&KeyProvider{}, &KeyConsumer{}).Handler()
				foo, _, err := miruken.Execute[*Foo](handler, Bar{})
				suite.Nil(err)
				suite.NotNil(foo)
				suite.Equal(4, foo.Count())
			})
		})
	})

	suite.Run("Open", func () {
		handler, _ := setup.New().Specs(&OpenProvider{}).Handler()
		foo, _, ok, err := miruken.Resolve[*Foo](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(1, foo.Count())
		foo, _, ok, err = miruken.Resolve[*Foo](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(1, foo.Count())
		bar, _, ok, err := miruken.Resolve[*Bar](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(2, bar.Count())
		bar, _, ok, err = miruken.Resolve[*Bar](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(2, bar.Count())
		foo, _, ok, err = miruken.ResolveKey[*Foo](handler, "Foo")
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(2, foo.Count())
		foo, _, ok, err = miruken.ResolveKey[*Foo](handler, "Foo")
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(2, foo.Count())
		foo, _, ok, err = miruken.Resolve[*Foo](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(2, foo.Count())
	})

	suite.Run("OpenScoped", func () {
		handler, _ := setup.New().Specs(&OpenProvider{}).Handler()
		ctx := context.New(handler)
		baz, _, ok, err := miruken.Resolve[*Baz](ctx)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(1, baz.Count())
		baz, _, ok, err = miruken.Resolve[*Baz](ctx)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(1, baz.Count())
		bam, _, ok, err := miruken.Resolve[*Bam](ctx)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(2, bam.Count())
		bam, _, ok, err = miruken.Resolve[*Bam](ctx)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(2, bam.Count())
		baz, _, ok, err = miruken.ResolveKey[*Baz](ctx, "Baz")
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(2, baz.Count())
		baz, _, ok, err = miruken.ResolveKey[*Baz](ctx, "Baz")
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(2, baz.Count())
		baz, _, ok, err = miruken.Resolve[*Baz](ctx)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(2, baz.Count())

		child := ctx.NewChild()
		baz, _, ok, err = miruken.Resolve[*Baz](child, provides.Explicit)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(3, baz.Count())
		baz, _, ok, err = miruken.Resolve[*Baz](child, provides.Explicit)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(3, baz.Count())
		bam, _, ok, err = miruken.Resolve[*Bam](child, provides.Explicit)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(4, bam.Count())
		bam, _, ok, err = miruken.Resolve[*Bam](child, provides.Explicit)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(4, bam.Count())
		baz, _, ok, err = miruken.ResolveKey[*Baz](child, "Baz", provides.Explicit)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(4, baz.Count())
		baz, _, ok, err = miruken.Resolve[*Baz](child, provides.Explicit)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(4, baz.Count())
	})

	suite.Run("Multiple", func () {
		handler, _ := setup.New().Specs(&MultiProvider{}).Handler()
		foo, _, ok, err := miruken.Resolve[*Foo](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(2, foo.Count())

		bar, _, ok, err := miruken.Resolve[*Bar](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.Equal(1, bar.Count())

		bar, _, ok, err = miruken.Resolve[*Bar](handler)
		suite.False(ok)
		suite.Nil(err)
		suite.Nil(bar)
	})

	suite.Run("Specification", func () {
		handler, _ := setup.New().Specs(&SpecificationProvider{}).Handler()
		handler = miruken.BuildUp(handler, provides.With(Baz{Counted{2}}))

		suite.Run("Invariant", func () {
			foo, _, ok, err := miruken.Resolve[*Foo](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.Equal(3, foo.Count())
		})

		suite.Run("Strict", func () {
			bar, _, ok, err := miruken.Resolve[*Bar](handler)
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(bar)

			bars, _, ok, err := miruken.Resolve[[]*Bar](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(bars)
			suite.Equal(2, len(bars))
		})
	})

	suite.Run("Lists", func () {
		handler, _ := setup.New().Specs(&ListProvider{}).Handler()

		suite.Run("Slice", func () {
			foo, _, ok, err := miruken.Resolve[*Foo](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(foo)
		})

		suite.Run("Array", func () {
			bar, _, ok, err := miruken.Resolve[*Bar](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(bar)
		})
	})

	suite.Run("Constructor", func () {
		handler, _ := suite.Setup()

		suite.Run("NoInit", func () {
			fooProvider, _, ok, err := miruken.Resolve[*FooProvider](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(fooProvider)
		})

		suite.Run("Constructor", func () {
			multiProvider, _, ok, err := miruken.Resolve[*MultiProvider](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(multiProvider)
			suite.Equal(1, multiProvider.foo.Count())
		})

		suite.Run("ConstructorDependencies", func () {
			handler, _ := setup.New().Specs(&SpecificationProvider{}).Handler()
			specProvider, _, ok, err := miruken.Resolve[*SpecificationProvider](
				miruken.BuildUp(handler, provides.With(Baz{Counted{2}})))
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(specProvider)
			suite.Equal(2, specProvider.foo.Count())
			suite.Equal(0, specProvider.bar.Count())
		})

		suite.Run("NoConstructor", func () {
			unmanaged, _, ok, err := miruken.Resolve[*UnmanagedHandler](handler)
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(unmanaged)
		})
	})

	suite.Run("Infer", func () {
		suite.Run("Invariant", func() {
			handler, _ := setup.New().Specs(&SpecificationHandler{}).Handler()
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
			// 1 from OpenProvider.Build
			// 1 from SimpleAsyncProvider.ProvideFoo
			// None from SpecificationProvider.ProvideFoo since it
			//   depends on an unsatisfied Baz
			// 5 total
			suite.Equal(6, len(foo))
		})

		suite.Run("Disable", func() {
			handler, _ := setup.New().
				WithoutInference().
				Handlers(new(FooProvider)).
				Handler()
			foo := new(Foo)
			result := handler.Handle(foo, false, nil)
			suite.False(result.IsError())
			suite.Equal(miruken.NotHandled, result)
		})
	})

	suite.Run("ResolveAll", func () {
		suite.Run("Invariant", func () {
			handler, _ := setup.New().
				Specs(
					&FooProvider{},
					&MultiProvider{},
					&SpecificationProvider{}).
				Handlers(
					new(FooProvider), new(MultiProvider), new (SpecificationProvider)).
				Handler()

			if foo, _, err := miruken.ResolveAll[*Foo](handler); err == nil {
				suite.NotNil(foo)
				// 2 for inference of *FooProvider (1) which includes explicit instance (1)
				// 2 for inference of *MultiProvider (1) which includes explicit instance (1)
				// 1 for inference of *SpecificationProvider (1) which excludes constructed
				//   instance since it has an unsatisfied dependency on Baz
				// 8 total
				suite.Len(foo, 5)
				suite.True(foo[0] != foo[1])
			} else {
				suite.Fail("unexpected error", err.Error())
			}
		})

		suite.Run("Covariant", func () {
			handler, _ := setup.New().
				Specs(&ListProvider{}).
				Handlers(new(ListProvider)).
				Handler()
			if counted, _, err := miruken.ResolveAll[Counter](handler); err == nil {
				suite.NotNil(counted)
				// 8 for inference of *ListProvider (4) which includes explicit instance (4)
				suite.Len(counted, 8)
			} else {
				suite.Fail("unexpected error", err.Error())
			}
		})

		suite.Run("Empty", func () {
			handler, _ := setup.New().Handlers(new(FooProvider)).Handler()
			bars, _, err := miruken.ResolveAll[*Bar](handler)
			suite.Nil(err)
			suite.Nil(bars)
		})
	})

	suite.Run("With", func () {
		handler, _ := setup.New().Handler()
		fooProvider, _, ok, err := miruken.Resolve[*FooProvider](handler)
		suite.False(ok)
		suite.Nil(err)
		suite.Nil(fooProvider)
		fooProvider, _, ok, err = miruken.Resolve[*FooProvider](
			miruken.BuildUp(handler, provides.With(new(FooProvider))))
		suite.True(ok)
		suite.Nil(err)
		suite.NotNil(fooProvider)
	})

	suite.Run("Invalid", func () {
		failures := 0
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(*miruken.HandlerInfoError); ok {
					var errMethod *miruken.MethodBindingError
					for cause := errors.Unwrap(err.Cause);
						errors.As(cause, &errMethod); cause = errors.Unwrap(cause) {
						failures++
					}
					suite.Equal(6, failures)
				} else {
					suite.Fail("Expected HandlerInfoError")
				}
			}
		}()
		_, err := setup.New().
			Specs(&InvalidProvider{}).
			Handlers(new(InvalidProvider)).
			Handler()
		suite.Nil(err)
		suite.Fail("should cause panic")
	})

	suite.Run("Function Binding", func () {
		suite.Run("Implied", func() {
			handler, _ := setup.New().Specs(ProvideBar).Handler()
			bar, _, ok, err := miruken.Resolve[*Bar](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(bar)
			suite.Equal(2, bar.Count())
		})
	})
}

func (suite *ProvidesTestSuite) TestProvidesAsync() {
	suite.Run("Simple", func () {
		suite.Run("Returns Promise", func() {
			handler, _ := setup.New().Specs(&SimpleAsyncProvider{}).Handler()
			foo, pf, ok, err := miruken.Resolve[*Foo](handler)
			suite.True(ok)
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
			handler, _ := setup.New().
				Specs(&SimpleAsyncProvider{}, &ComplexAsyncProvider{}).
				Handler()
			bar, pb, ok, err := miruken.Resolve[*Bar](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.Nil(bar)
			suite.NotNil(pb)
			bar, err = pb.Await()
			suite.Nil(err)
			suite.Equal(3, bar.Count())
			foo, pf, ok, err := miruken.Resolve[*Foo](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.Nil(foo)
			suite.NotNil(pf)
			foo, err = pf.Await()
			suite.Nil(err)
			suite.Equal(3, foo.Count())
		})

		suite.Run("Async Args", func() {
			handler, _ := setup.New().
				Specs(
					&AsyncArgProvider{},
					&ComplexAsyncProvider{},
					&SimpleAsyncProvider{}).
				Handler()
			bar, pb, ok, err := miruken.Resolve[*Bar](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.Nil(bar)
			suite.NotNil(pb)
			bar, err = pb.Await()
			suite.Nil(err)
			suite.Equal(3, bar.Count())
			foo, pf, ok, err := miruken.Resolve[*Foo](handler)
			suite.True(ok)
			suite.Nil(err)
			suite.Nil(foo)
			suite.NotNil(pf)
			foo, err = pf.Await()
			suite.Nil(err)
			suite.Equal(7, foo.Count())
		})
	})
}

func TestProvidesTestSuite(t *testing.T) {
	suite.Run(t, new(ProvidesTestSuite))
}
