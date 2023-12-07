package test

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
	"testing"
)

type (
	Counter interface {
		Count() int
		Inc() int
	}

	Counted struct {
		count int
	}

	Foo struct { Counted }
	Bar struct { Counted }
)

func (c *Counted) Count() int {
	return c.count
}

func (c *Counted) Inc() int {
	c.count++
	return c.count
}

type ContextObserver struct {
	contextEnding bool
	contextEndingReason any
	contextEnded bool
	contextEndedReason any
	childContextEnding bool
	childContextEndingContext *context.Context
	childContextEndingReason any
	childContextEnded bool
	childContextEndedContext *context.Context
	childContextEndedReason any
}

func (o *ContextObserver) ContextEnding(
	ctx    *context.Context,
	reason  any,
) {
	o.contextEnding       = true
	o.contextEndingReason = reason
}

func (o *ContextObserver) ContextEnded(
	ctx    *context.Context,
	reason  any,
) {
	o.contextEnded       = true
	o.contextEndedReason = reason
}

func (o *ContextObserver) ChildContextEnding(
	childCtx *context.Context,
	reason    any,
) {
	o.childContextEnding        = true
	o.childContextEndingContext = childCtx
	o.childContextEndingReason  = reason
}

func (o *ContextObserver) ChildContextEnded(
	childCtx *context.Context,
	reason    any,
) {
	o.childContextEnded        = true
	o.childContextEndedContext = childCtx
	o.childContextEndedReason  = reason
}

type Service struct {}

func (s *Service) Count(
	_ *handles.It, counter Counter,
) {
	counter.Inc()
}

type ContextTestSuite struct {
	suite.Suite
	HandleTypes []any
}

func (suite *ContextTestSuite) SetupTest() {
	handleTypes := []any{
		&Service{},
		&ScopedService{},
		&RootedService{},
	}
	suite.HandleTypes = handleTypes
}

func (suite *ContextTestSuite) RootContext() *context.Context {
	return suite.RootContextWith(suite.HandleTypes...)
}

func (suite *ContextTestSuite) RootContextWith(specs ...any) *context.Context {
	ctx, _ := setup.New().Specs(specs...).Context()
	return ctx
}

func (suite *ContextTestSuite) TestContext() {
	suite.Run("InitiallyActive", func () {
		ctx := context.New()
		suite.Equal(context.StateActive, ctx.State())
	})

	suite.Run("GetSelf", func () {
		ctx := context.New()
		self, _, ok, err := provides.Type[*context.Context](ctx)
		suite.True(ok)
		suite.Nil(err)
		suite.Same(ctx, self)
	})

	suite.Run("RootNoParent", func () {
		ctx := context.New()
		suite.Nil(ctx.Parent())
	})

	suite.Run("GetRootContext", func () {
		ctx   := context.New()
		child := ctx.NewChild()
		suite.Same(ctx, ctx.Root())
		suite.Same(ctx, child.Root())
	})

	suite.Run("GetParenContext", func () {
		ctx   := context.New()
		child := ctx.NewChild()
		suite.Same(ctx, child.Parent())
	})

	suite.Run("NoChildrenByDefault", func () {
		ctx := context.New()
		suite.Nil(ctx.Children())
	})

	suite.Run("ChildrenAvailable", func () {
		ctx    := context.New()
		child1 := ctx.NewChild()
		child2 := ctx.NewChild()
		suite.True(ctx.HasChildren())
		suite.ElementsMatch(ctx.Children(), []*context.Context{child1, child2})
	})

	suite.Run("Handlers", func () {
		handler, _ := setup.New().
			WithoutInference().
			Specs(&Service{}).
			Context()
		ctx := context.New(handler, new(Service))
		var foo Foo
		result := ctx.Handle(&foo, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
		suite.Equal(1, foo.Count())
	})

	suite.Run("End", func () {
		ctx := context.New()
		ctx.End(nil)
		suite.Equal(context.StateEnded, ctx.State())
	})

	suite.Run("EndChild", func () {
		ctx   := context.New()
		child := ctx.NewChild()
		ctx.End(nil)
		suite.Equal(context.StateEnded, child.State())
	})

	suite.Run("EndWhenDisposed", func () {
		ctx := context.New()
		ctx.Dispose()
		suite.Equal(context.StateEnded, ctx.State())
	})

	suite.Run("Unwind", func () {
		ctx    := context.New()
		child1 := ctx.NewChild()
		child2 := ctx.NewChild()
		ctx.Unwind(nil)
		suite.Equal(context.StateEnded, child1.State())
		suite.Equal(context.StateEnded, child2.State())
	})

	suite.Run("UnwindRoot", func () {
		ctx        := context.New()
		child1     := ctx.NewChild()
		child2     := ctx.NewChild()
		grandChild := child1.NewChild()
		root       := child2.UnwindToRoot(nil)
		suite.Same(ctx, root)
		suite.Equal(context.StateActive, ctx.State())
		suite.Equal(context.StateEnded, child1.State())
		suite.Equal(context.StateEnded, child2.State())
		suite.Equal(context.StateEnded, grandChild.State())
	})

	suite.Run("Store", func () {
		data := &ContextObserver{}
		ctx  := context.New()
		ctx.Store(data)
		resolve, _, ok, err := provides.Type[*ContextObserver](ctx)
		suite.True(ok)
		suite.Nil(err)
		suite.Same(data, resolve)
	})

	suite.Run("Traverse", func () {
		suite.Run("AncestorsByDefault", func() {
			data  := &ContextObserver{}
			root  := context.New()
			child := root.NewChild()
			grandChild := child.NewChild()
			root.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](grandChild)
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("Self", func() {
			data  := &ContextObserver{}
			root  := context.New()
			child := root.NewChild()
			root.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(child, miruken.SelfAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(root, miruken.SelfAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("Root", func() {
			data  := &ContextObserver{}
			root  := context.New()
			child := root.NewChild()
			child.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(child, miruken.RootAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			root.Store(data)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(root, miruken.RootAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("Children", func() {
			data := &ContextObserver{}
			root := context.New()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			child2.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(child2, miruken.ChildAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(grandChild, miruken.ChildAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(root, miruken.ChildAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("Siblings", func() {
			data := &ContextObserver{}
			root := context.New()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			child3.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(root, miruken.SiblingAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(child3, miruken.SiblingAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(grandChild, miruken.SiblingAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(child2, miruken.SiblingAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("ChildrenOrSelf", func() {
			data   := &ContextObserver{}
			root   := context.New()
			child1 := root.NewChild()
			root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			child3.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(child1, miruken.SelfOrChildAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(grandChild, miruken.SelfOrChildAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(child3, miruken.SelfOrChildAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(root, miruken.SelfOrChildAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("SiblingsOrSelf", func() {
			data := &ContextObserver{}
			root := context.New()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			child3.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(root, miruken.SelfOrSiblingAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(grandChild, miruken.SelfOrSiblingAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(child3, miruken.SelfOrSiblingAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(child2, miruken.SelfOrSiblingAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("Ancestors", func() {
			data  := &ContextObserver{}
			root  := context.New()
			child := root.NewChild()
			grandChild := child.NewChild()
			root.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(root, miruken.AncestorAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(grandChild, miruken.AncestorAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("AncestorsOrSelf", func() {
			data  := &ContextObserver{}
			root  := context.New()
			child := root.NewChild()
			grandChild := child.NewChild()
			root.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(root, miruken.SelfOrAncestorAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(grandChild, miruken.SelfOrAncestorAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("Descendants", func() {
			data := &ContextObserver{}
			root := context.New()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			grandChild.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(grandChild, miruken.DescendantAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(child2, miruken.DescendantAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(child3, miruken.DescendantAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(root, miruken.DescendantAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("DescendantsReverse", func() {
			data := &ContextObserver{}
			root := context.New()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			grandChild.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(grandChild, miruken.DescendantReverseAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(child2, miruken.DescendantReverseAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(child3, miruken.DescendantReverseAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(root, miruken.DescendantReverseAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("DescendantsOrSelf", func() {
			data := &ContextObserver{}
			root := context.New()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			grandChild.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(child2, miruken.SelfOrDescendantAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(grandChild, miruken.SelfOrDescendantAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(child3, miruken.SelfOrDescendantAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(root, miruken.SelfOrDescendantAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("DescendantsOrSelfReverse", func() {
			data := &ContextObserver{}
			root := context.New()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			grandChild.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(child2, miruken.SelfOrDescendantReverseAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(grandChild, miruken.SelfOrDescendantReverseAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(child3, miruken.SelfOrDescendantReverseAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(root, miruken.SelfOrDescendantReverseAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("DescendantsOrSELF", func() {
			data := &ContextObserver{}
			root := context.New()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			child3.NewChild()
			root.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(child2, miruken.SelfOrDescendantAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(root, miruken.SelfOrDescendantAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("AncestorsSIBLINGSorSelf", func() {
			data := &ContextObserver{}
			root := context.New()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			child2.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(grandChild, miruken.SelfSiblingOrAncestorAxis))
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(resolve)
			resolve, _, ok, err = provides.Type[*ContextObserver](miruken.BuildUp(child3, miruken.SelfSiblingOrAncestorAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("ANCESTORSSiblingsOrSelf", func() {
			data := &ContextObserver{}
			root := context.New()
			root.NewChild()
			root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			child3.Store(data)
			resolve, _, ok, err := provides.Type[*ContextObserver](miruken.BuildUp(grandChild, miruken.SelfSiblingOrAncestorAxis))
			suite.True(ok)
			suite.Nil(err)
			suite.Same(data, resolve)
		})
	})

	suite.Run("Observe", func () {
		suite.Run("StateEnding", func() {
			reason   := "ending"
			observer := ContextObserver{}
			ctx      := context.New()
			ctx.Observe(&observer)
			ctx.End(reason)
			suite.True(observer.contextEnding)
			suite.Equal(reason, observer.contextEndingReason)
		})

		suite.Run("StateEnded", func() {
			reason   := "ended"
			observer := ContextObserver{}
			ctx      := context.New()
			ctx.Observe(&observer)
			ctx.End(reason)
			suite.True(observer.contextEnded)
			suite.Equal(reason, observer.contextEndedReason)
		})

		suite.Run("ChildContextEnding", func() {
			reason   := "childEnding"
			observer := ContextObserver{}
			ctx      := context.New()
			ctx.Observe(&observer)
			child := ctx.NewChild()
			child.End(reason)
			suite.False(observer.contextEnding)
			suite.True(observer.childContextEnding)
			suite.Same(child, observer.childContextEndingContext)
			suite.Equal(reason, observer.childContextEndingReason)
		})

		suite.Run("ChildContextEnded", func() {
			reason   := "childEnded"
			observer := ContextObserver{}
			ctx      := context.New()
			ctx.Observe(&observer)
			child := ctx.NewChild()
			child.End(reason)
			suite.False(observer.contextEnded)
			suite.True(observer.childContextEnded)
			suite.Same(child, observer.childContextEndedContext)
			suite.Equal(reason, observer.childContextEndedReason)
		})
	})
}

// ScopedService demonstrates a scoped service.
type ScopedService struct {
	context.ContextualBase
	disposed bool
	foo      Foo
}

func (s *ScopedService) Constructor(
	_*struct{
		provides.It
		context.Scoped
	  },
) {
}

func (s *ScopedService) SetContext(ctx *context.Context) {
	s.ChangeContext(s, ctx)
}

func (s *ScopedService) Count(
	_ *handles.It, counter Counter,
) {
	if s.Context() == nil {
		panic("context not assigned")
	}
	counter.Inc()
}

func (s *ScopedService) ProvideFoo(*provides.It) *Foo {
	if s.Context() == nil {
		panic("context not assigned")
	}
	s.foo.Inc()
	return &s.foo
}

func (s *ScopedService) Dispose() {
	s.disposed = true
}

// RootedService demonstrates a root scoped service.
type RootedService struct {
	context.ContextualBase
	disposed bool
	bar      Bar
}

func (s *RootedService) Constructor(
	_*struct{
		provides.It
		context.Rooted
      },
) {
}

func (s *RootedService) SetContext(ctx *context.Context) {
	s.ChangeContext(s, ctx)
}

func (s *RootedService) HandleBar(
	_ *handles.It, bar *Bar,
) {
	if s.Context() == nil {
		panic("context not assigned")
	}
	bar.Inc()
	bar.Inc()
}

func (s *RootedService) ProvideCounter(*provides.It) Counter {
	if s.Context() == nil {
		panic("context not assigned")
	}
	s.bar.Inc()
	s.bar.Inc()
	return &s.bar
}

func (s *RootedService) Dispose() {
	s.disposed = true
}

type LifestyleMismatch struct {}

func (l *LifestyleMismatch) Constructor(
	_*struct{
		provides.It
		provides.Single
	  },
	service *ScopedService,
) {
}

// ContextualObserver collects Contextual changes.
type ContextualObserver struct {
	contextual [2]context.Contextual
	oldCtx     [2]*context.Context
	newCtx     [2]*context.Context
	useCtx     *context.Context
}

func (o *ContextualObserver) ContextChanging(
	contextual context.Contextual,
	oldCtx      *context.Context,
	newCtx     **context.Context,
) {
	o.contextual[0] = contextual
	o.oldCtx[0]     = oldCtx
	o.newCtx[0]     = *newCtx
	if o.useCtx != nil {
		*newCtx = o.useCtx
	}
}

func (o *ContextualObserver) ContextChanged(
	contextual context.Contextual,
	oldCtx     *context.Context,
	newCtx     *context.Context,
) {
	o.contextual[1] = contextual
	o.oldCtx[1]     = oldCtx
	o.newCtx[1]     = newCtx
}

func (suite *ContextTestSuite) TestContextual() {
	suite.Run("ContextInitiallyEmpty", func () {
		service := ScopedService{}
		suite.Nil(service.Context())
	})

	suite.Run("SetContext", func () {
		service := ScopedService{}
		root    := context.New()
		service.SetContext(root)
		suite.Same(root, service.Context())
	})

	suite.Run("AddsContextualToContext", func () {
		service := ScopedService{}
		root    := context.New()
		service.SetContext(root)
		if services, _, err := provides.All[*ScopedService](root); err == nil {
			suite.NotNil(services)
			suite.Len(services, 1)
			suite.Same(&service, services[0])
		} else {
			suite.Fail("unexpected error: %v", err.Error())
		}
	})

	suite.Run("RemovesContextualFromContext", func () {
		service := ScopedService{}
		root    := context.New()
		service.SetContext(root)
		if services, _, err := provides.All[*ScopedService](root); err == nil {
			suite.NotNil(services)
			suite.Len(services, 1)
			suite.Same(&service, services[0])
		} else {
			suite.Fail("unexpected error: %v", err.Error())
		}
		service.SetContext(nil)
		if services, _, err := provides.All[*ScopedService](root); err == nil {
			suite.Nil(services)
			suite.Len(services, 0)
		} else {
			suite.Fail("unexpected error: %v", err.Error())
		}
	})

	suite.Run("Observes", func () {
		suite.Run("SetContext", func() {
			service  := ScopedService{}
			observer := ContextualObserver{}
			service.Observe(&observer)
			root := context.New()
			service.SetContext(root)
			suite.Same(root, service.Context())
			suite.Same(&service, observer.contextual[0])
			suite.Same(&service, observer.contextual[1])
			suite.Nil(observer.oldCtx[0])
			suite.Nil(observer.oldCtx[1])
			suite.Same(root, observer.newCtx[0])
			suite.Same(root, observer.newCtx[1])
		})

		suite.Run("ClearContext", func() {
			service := ScopedService{}
			root    := context.New()
			service.SetContext(root)
			observer := ContextualObserver{}
			service.Observe(&observer)
			service.SetContext(nil)
			suite.Nil(service.Context())
			suite.Same(&service, observer.contextual[0])
			suite.Same(&service, observer.contextual[1])
			suite.Same(root, observer.oldCtx[0])
			suite.Same(root, observer.oldCtx[1])
			suite.Nil(observer.newCtx[0])
			suite.Nil(observer.newCtx[1])
		})

		suite.Run("ReplaceContext", func () {
			service  := ScopedService{}
			root     := context.New()
			child    := root.NewChild()
			observer := ContextualObserver{useCtx: child}
			service.Observe(&observer)
			service.SetContext(root)
			suite.Same(child, service.Context())
			suite.Same(&service, observer.contextual[0])
			suite.Same(&service, observer.contextual[1])
			suite.Nil(observer.oldCtx[0])
			suite.Nil(observer.oldCtx[1])
			suite.Same(root, observer.newCtx[0])
			suite.Same(child, observer.newCtx[1])
		})
	})

	suite.Run("Resolve", func () {
		suite.Run("ContextAssigned", func() {
			root := suite.RootContext()
			service, _, ok, err := provides.Type[*ScopedService](root)
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(service)
			suite.Same(root, service.Context())
			suite.False(service.disposed)
			service2, _,ok,  err := provides.Type[*ScopedService](root)
			suite.True(ok)
			suite.Nil(err)
			suite.Same(service, service2)
		})

		suite.Run("SameContextualWithoutQualifier", func() {
			root  := suite.RootContext()
			child := root.NewChild()
			service, _, ok, err := provides.Type[*ScopedService](root)
			suite.True(ok)
			suite.Nil(err)
			suite.Same(root, service.Context())
			childService, _, ok, err := provides.Type[*ScopedService](child)
			suite.True(ok)
			suite.Nil(err)
			suite.Same(service, childService)
		})

		suite.Run("ChildContextAssigned", func() {
			root  := suite.RootContext()
			child := root.NewChild()
			service, _, ok, err := provides.Type[*ScopedService](root)
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(service)
			childService, _, ok, err := provides.Type[*ScopedService](child, provides.Explicit)
			suite.True(ok)
			suite.Nil(err)
			suite.Same(child, childService.Context())
			suite.NotSame(service, childService)
			suite.False(childService.disposed)
			child.End(nil)
			suite.True(childService.disposed)
			suite.False(service.disposed)
			root.End(nil)
			suite.True(service.disposed)
		})

		suite.Run("DisposedWhenContextEnds", func() {
			root := suite.RootContext()
			service, _, ok, err := provides.Type[*ScopedService](root)
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(service)
			suite.Same(root, service.Context())
			suite.False(service.disposed)
			root.End(nil)
			suite.Nil(service.Context())
			suite.True(service.disposed)
			service2, _, ok, err := provides.Type[*ScopedService](root)
			suite.False(ok)
			suite.NotNil(err)
			suite.Equal("scoped: cannot scope instances to an inactive context", err.Error())
			suite.Nil(service2)
		})

		suite.Run("UnmanagedWhenContextNil", func() {
			root := suite.RootContext()
			service, _, ok, err := provides.Type[*ScopedService](root)
			suite.True(ok)
			suite.Nil(err)
			service.SetContext(nil)
			suite.True(service.disposed)
			service2, _, ok, err := provides.Type[*ScopedService](root)
			suite.True(ok)
			suite.Nil(err)
			suite.NotSame(service, service2)
		})

		suite.Run("RootContextAssigned", func() {
			root   := suite.RootContext()
			child1 := root.NewChild()
			child2 := root.NewChild()
			service, _, ok, err := provides.Type[*RootedService](child1)
			suite.True(ok)
			suite.Nil(err)
			suite.NotNil(service)
			suite.Same(root, service.Context())
			suite.False(service.disposed)
			service2, _, ok, err := provides.Type[*RootedService](child2)
			suite.True(ok)
			suite.Nil(err)
			suite.Same(service, service2)
		})

		suite.Run("FailIfContextChangedNonNil", func() {

		})

		suite.Run("FailIfContextChangedNonNil", func() {
			defer func() {
				if r := recover(); r != nil {
					if reason, ok := r.(string); ok {
						suite.Equal("managed instances cannot change context", reason)
					} else {
						suite.Fail("Expected managed instances cannot change context")
					}
				}
			}()
			root := suite.RootContext()
			service, _, ok, err := provides.Type[*ScopedService](root)
			suite.True(ok)
			suite.Nil(err)
			service.SetContext(root.NewChild())
		})
	})

	suite.Run("Infer", func () {
		suite.Run("Handles", func () {
			suite.Run("Invariant", func() {
				root   := suite.RootContext()
				bar    := new(Bar)
				result := root.Handle(bar, false, nil)
				suite.False(result.IsError())
				suite.Equal(miruken.Handled, result)
				suite.Equal(2, bar.Count())
			})

			suite.Run("Contravariant", func() {
				root   := suite.RootContext()
				foo    := new(Foo)
				result := root.Handle(foo, false, nil)
				suite.False(result.IsError())
				suite.Equal(miruken.Handled, result)
				suite.Equal(1, foo.Count())
			})
		})

		suite.Run("Build", func () {
			suite.Run("Invariant", func() {
				root := suite.RootContext()
				foo, _, ok, err := provides.Type[*Foo](root)
				suite.True(ok)
				suite.Nil(err)
				suite.Equal(1, foo.Count())
			})

			suite.Run("Covariant", func() {
				root := suite.RootContextWith(&RootedService{})
				counter, _, ok, err := provides.Type[Counter](root)
				suite.True(ok)
				suite.Nil(err)
				suite.Equal(2, counter.Count())
			})
		})

		suite.Run("RejectScopedDependencyInSingleton", func() {
			root := suite.RootContextWith(
				&ScopedService{},
				&LifestyleMismatch{})
			mismatch, _, ok, err := provides.Type[*LifestyleMismatch](root)
			suite.False(ok)
			suite.Nil(err)
			suite.Nil(mismatch)
		})
	})
}

func TestContextTestSuite(t *testing.T) {
	suite.Run(t, new(ContextTestSuite))
}
