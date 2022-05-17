package test

import (
	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ContextObserver struct {
	contextEnding bool
	contextEndingReason any
	contextEnded bool
	contextEndedReason any
	childContextEnding bool
	childContextEndingContext *miruken.Context
	childContextEndingReason any
	childContextEnded bool
	childContextEndedContext *miruken.Context
	childContextEndedReason any
}

func (o *ContextObserver) ContextEnding(
	ctx    *miruken.Context,
	reason  any,
) {
	o.contextEnding       = true
	o.contextEndingReason = reason
}

func (o *ContextObserver) ContextEnded(
	ctx    *miruken.Context,
	reason  any,
) {
	o.contextEnded       = true
	o.contextEndedReason = reason
}

func (o *ContextObserver) ChildContextEnding(
	childCtx *miruken.Context,
	reason    any,
) {
	o.childContextEnding        = true
	o.childContextEndingContext = childCtx
	o.childContextEndingReason  = reason
}

func (o *ContextObserver) ChildContextEnded(
	childCtx *miruken.Context,
	reason    any,
) {
	o.childContextEnded        = true
	o.childContextEndedContext = childCtx
	o.childContextEndedReason  = reason
}

type Service struct {}

func (s *Service) Count(
	_ *miruken.Handles, counter Counter,
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

func (suite *ContextTestSuite) RootContext() *miruken.Context {
	return suite.RootContextWith(suite.HandleTypes...)
}

func (suite *ContextTestSuite) RootContextWith(specs ... any) *miruken.Context {
	return miruken.NewContext(miruken.Setup(miruken.WithHandlerSpecs(specs...)))
}

func (suite *ContextTestSuite) TestContext() {
	suite.Run("InitiallyActive", func () {
		context := miruken.NewContext()
		suite.Equal(miruken.ContextActive, context.State())
	})

	suite.Run("GetSelf", func () {
		context := miruken.NewContext()
		var self *miruken.Context
		err := miruken.Resolve(context, &self)
		suite.Nil(err)
		suite.Same(context, self)
	})

	suite.Run("RootNoParent", func () {
		context := miruken.NewContext()
		suite.Nil(context.Parent())
	})

	suite.Run("GetRootContext", func () {
		context := miruken.NewContext()
		child   := context.NewChild()
		suite.Same(context, context.Root())
		suite.Same(context, child.Root())
	})

	suite.Run("GetParenContext", func () {
		context := miruken.NewContext()
		child   := context.NewChild()
		suite.Same(context, child.Parent())
	})

	suite.Run("NoChildrenByDefault", func () {
		context := miruken.NewContext()
		suite.Nil(context.Children())
	})

	suite.Run("ChildrenAvailable", func () {
		context := miruken.NewContext()
		child1  := context.NewChild()
		child2  := context.NewChild()
		suite.True(context.HasChildren())
		suite.ElementsMatch(context.Children(), []*miruken.Context{child1, child2})
	})

	suite.Run("Handlers", func () {
		context := miruken.NewContext(miruken.Setup(
			miruken.WithHandlerSpecs(&Service{}),
			miruken.WithoutInference,
		), new(Service))
		var foo Foo
		result := context.Handle(&foo, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
		suite.Equal(1, foo.Count())
	})

	suite.Run("End", func () {
		context := miruken.NewContext()
		context.End(nil)
		suite.Equal(miruken.ContextEnded, context.State())
	})

	suite.Run("EndChild", func () {
		context := miruken.NewContext()
		child   := context.NewChild()
		context.End(nil)
		suite.Equal(miruken.ContextEnded, child.State())
	})

	suite.Run("EndWhenDisposed", func () {
		context := miruken.NewContext()
		context.Dispose()
		suite.Equal(miruken.ContextEnded, context.State())
	})

	suite.Run("Unwind", func () {
		context := miruken.NewContext()
		child1  := context.NewChild()
		child2  := context.NewChild()
		context.Unwind(nil)
		suite.Equal(miruken.ContextEnded, child1.State())
		suite.Equal(miruken.ContextEnded, child2.State())
	})

	suite.Run("UnwindRoot", func () {
		context    := miruken.NewContext()
		child1     := context.NewChild()
		child2     := context.NewChild()
		grandChild := child1.NewChild()
		root       := child2.UnwindToRoot(nil)
		suite.Same(context, root)
		suite.Equal(miruken.ContextActive, context.State())
		suite.Equal(miruken.ContextEnded, child1.State())
		suite.Equal(miruken.ContextEnded, child2.State())
		suite.Equal(miruken.ContextEnded, grandChild.State())
	})

	suite.Run("Store", func () {
		data := &ContextObserver{}
		context := miruken.NewContext()
		context.Store(data)
		var resolve *ContextObserver
		err := miruken.Resolve(context, &resolve)
		suite.Nil(err)
		suite.Same(data, resolve)
	})

	suite.Run("Traverse", func () {
		suite.Run("AncestorsByDefault", func() {
			data := &ContextObserver{}
			root := miruken.NewContext()
			child := root.NewChild()
			grandChild := child.NewChild()
			root.Store(data)
			var resolve *ContextObserver
			err := miruken.Resolve(grandChild, &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("Self", func() {
			data := &ContextObserver{}
			root := miruken.NewContext()
			child := root.NewChild()
			root.Store(data)
			var resolve *ContextObserver
			err := miruken.Resolve(miruken.Build(child, miruken.WithSelf), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(root, miruken.WithSelf), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("Root", func() {
			data := &ContextObserver{}
			root := miruken.NewContext()
			child := root.NewChild()
			child.Store(data)
			var resolve *ContextObserver
			err := miruken.Resolve(miruken.Build(child, miruken.WithRoot), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			root.Store(data)
			err = miruken.Resolve(miruken.Build(root, miruken.WithRoot), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("Children", func() {
			data := &ContextObserver{}
			root := miruken.NewContext()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			child2.Store(data)
			var resolve *ContextObserver
			err := miruken.Resolve(miruken.Build(child2, miruken.WithChild), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(grandChild, miruken.WithChild), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(root, miruken.WithChild), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("Siblings", func() {
			data := &ContextObserver{}
			root := miruken.NewContext()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			child3.Store(data)
			var resolve *ContextObserver
			err := miruken.Resolve(miruken.Build(root, miruken.WithSibling), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(child3, miruken.WithSibling), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(grandChild, miruken.WithSibling), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(child2, miruken.WithSibling), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("ChildrenOrSelf", func() {
			data := &ContextObserver{}
			root := miruken.NewContext()
			child1 := root.NewChild()
			root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			child3.Store(data)
			var resolve *ContextObserver
			err := miruken.Resolve(miruken.Build(child1, miruken.WithSelfOrChild), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(grandChild, miruken.WithSelfOrChild), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(child3, miruken.WithSelfOrChild), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
			err = miruken.Resolve(miruken.Build(root, miruken.WithSelfOrChild), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("SiblingsOrSelf", func() {
			data := &ContextObserver{}
			root := miruken.NewContext()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			child3.Store(data)
			var resolve *ContextObserver
			err := miruken.Resolve(miruken.Build(root, miruken.WithSelfOrSibling), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(grandChild, miruken.WithSelfOrSibling), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(child3, miruken.WithSelfOrSibling), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
			err = miruken.Resolve(miruken.Build(child2, miruken.WithSelfOrSibling), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("Ancestors", func() {
			data := &ContextObserver{}
			root := miruken.NewContext()
			child := root.NewChild()
			grandChild := child.NewChild()
			root.Store(data)
			var resolve *ContextObserver
			err := miruken.Resolve(miruken.Build(root, miruken.WithAncestor), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(grandChild, miruken.WithAncestor), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("AncestorsOrSelf", func() {
			data := &ContextObserver{}
			root := miruken.NewContext()
			child := root.NewChild()
			grandChild := child.NewChild()
			root.Store(data)
			var resolve *ContextObserver
			err := miruken.Resolve(miruken.Build(root, miruken.WithSelfOrAncestor), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
			err = miruken.Resolve(miruken.Build(grandChild, miruken.WithSelfOrAncestor), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("Descendants", func() {
			data := &ContextObserver{}
			root := miruken.NewContext()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			grandChild.Store(data)
			var resolve *ContextObserver
			err := miruken.Resolve(miruken.Build(grandChild, miruken.WithDescendant), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(child2, miruken.WithDescendant), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(child3, miruken.WithDescendant), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
			err = miruken.Resolve(miruken.Build(root, miruken.WithDescendant), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("DescendantsOrSelf", func() {
			data := &ContextObserver{}
			root := miruken.NewContext()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			grandChild.Store(data)
			var resolve *ContextObserver
			err := miruken.Resolve(miruken.Build(child2, miruken.WithSelfOrDescendant), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(grandChild, miruken.WithSelfOrDescendant), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
			err = miruken.Resolve(miruken.Build(child3, miruken.WithSelfOrDescendant), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
			err = miruken.Resolve(miruken.Build(root, miruken.WithSelfOrDescendant), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("DescendantsOrSELF", func() {
			data := &ContextObserver{}
			root := miruken.NewContext()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			child3.NewChild()
			root.Store(data)
			var resolve *ContextObserver
			err := miruken.Resolve(miruken.Build(child2, miruken.WithSelfOrDescendant), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(root, miruken.WithSelfOrDescendant), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("AncestorsSIBLINGSorSelf", func() {
			data := &ContextObserver{}
			root := miruken.NewContext()
			root.NewChild()
			child2 := root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			child2.Store(data)
			var resolve *ContextObserver
			err := miruken.Resolve(miruken.Build(grandChild, miruken.WithSelfSiblingOrAncestor), &resolve)
			suite.Nil(err)
			suite.Nil(resolve)
			err = miruken.Resolve(miruken.Build(child3, miruken.WithSelfSiblingOrAncestor), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
		})

		suite.Run("ANCESTORSSiblingsOrSelf", func() {
			data := &ContextObserver{}
			root := miruken.NewContext()
			root.NewChild()
			root.NewChild()
			child3 := root.NewChild()
			grandChild := child3.NewChild()
			child3.Store(data)
			var resolve *ContextObserver
			err := miruken.Resolve(miruken.Build(grandChild, miruken.WithSelfSiblingOrAncestor), &resolve)
			suite.Nil(err)
			suite.Same(data, resolve)
		})
	})

	suite.Run("Observe", func () {
		suite.Run("ContextEnding", func() {
			reason := "ending"
			observer := ContextObserver{}
			context := miruken.NewContext()
			context.Observe(&observer)
			context.End(reason)
			suite.True(observer.contextEnding)
			suite.Equal(reason, observer.contextEndingReason)
		})

		suite.Run("ContextEnded", func() {
			reason := "ended"
			observer := ContextObserver{}
			context := miruken.NewContext()
			context.Observe(&observer)
			context.End(reason)
			suite.True(observer.contextEnded)
			suite.Equal(reason, observer.contextEndedReason)
		})

		suite.Run("ChildContextEnding", func() {
			reason := "childEnding"
			observer := ContextObserver{}
			context := miruken.NewContext()
			context.Observe(&observer)
			child := context.NewChild()
			child.End(reason)
			suite.False(observer.contextEnding)
			suite.True(observer.childContextEnding)
			suite.Same(child, observer.childContextEndingContext)
			suite.Equal(reason, observer.childContextEndingReason)
		})

		suite.Run("ChildContextEnded", func() {
			reason := "childEnded"
			observer := ContextObserver{}
			context := miruken.NewContext()
			context.Observe(&observer)
			child := context.NewChild()
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
	miruken.ContextualBase
	disposed bool
	foo Foo
}

func (s *ScopedService) Constructor(
	_ *struct{
		miruken.Provides
		miruken.Scoped
	  },
) {
}

func (s *ScopedService) SetContext(ctx *miruken.Context) {
	s.ChangeContext(s, ctx)
}

func (s *ScopedService) Count(
	_ *miruken.Handles, counter Counter,
) {
	if s.Context() == nil {
		panic("context not assigned")
	}
	counter.Inc()
}

func (s *ScopedService) ProvideFoo(*miruken.Provides) *Foo {
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
	miruken.ContextualBase
	disposed bool
	bar Bar
}

func (s *RootedService) Constructor(
	_ *struct{
	     miruken.Provides
	     miruken.Scoped `scoped:"rooted"`
      },
) {
}

func (s *RootedService) SetContext(ctx *miruken.Context) {
	s.ChangeContext(s, ctx)
}

func (s *RootedService) HandleBar(
	_ *miruken.Handles, bar *Bar,
) {
	if s.Context() == nil {
		panic("context not assigned")
	}
	bar.Inc()
	bar.Inc()
}

func (s *RootedService) ProvideCounter(*miruken.Provides) Counter {
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
	_ *struct{
		miruken.Provides
		miruken.Singleton
	  },
	service *ScopedService,
) {
}

// ContextualObserver collects Contextual changes.
type ContextualObserver struct {
	contextual [2]miruken.Contextual
	oldCtx     [2]*miruken.Context
	newCtx     [2]*miruken.Context
	useCtx     *miruken.Context
}

func (o *ContextualObserver) ContextChanging(
	contextual   miruken.Contextual,
	oldCtx      *miruken.Context,
	newCtx     **miruken.Context,
) {
	o.contextual[0] = contextual
	o.oldCtx[0]     = oldCtx
	o.newCtx[0]     = *newCtx
	if o.useCtx != nil {
		*newCtx = o.useCtx
	}
}

func (o *ContextualObserver) ContextChanged(
	contextual  miruken.Contextual,
	oldCtx     *miruken.Context,
	newCtx     *miruken.Context,
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
		root    := miruken.NewContext()
		service.SetContext(root)
		suite.Same(root, service.Context())
	})

	suite.Run("AddsContextualToContext", func () {
		service := ScopedService{}
		root    := miruken.NewContext()
		service.SetContext(root)
		var services []*ScopedService
		if err := miruken.ResolveAll(root, &services); err == nil {
			suite.NotNil(services)
			suite.Len(services, 1)
			suite.Same(&service, services[0])
		} else {
			suite.Fail("unexpected error: %v", err.Error())
		}
	})

	suite.Run("RemovesContextualFromContext", func () {
		service := ScopedService{}
		root    := miruken.NewContext()
		service.SetContext(root)
		var services []*ScopedService
		if err := miruken.ResolveAll(root, &services); err == nil {
			suite.NotNil(services)
			suite.Len(services, 1)
			suite.Same(&service, services[0])
		} else {
			suite.Fail("unexpected error: %v", err.Error())
		}
		service.SetContext(nil)
		if err := miruken.ResolveAll(root, &services); err == nil {
			suite.NotNil(services)
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
			root := miruken.NewContext()
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
			root    := miruken.NewContext()
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
			root     := miruken.NewContext()
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
			var service *ScopedService
			err := miruken.Resolve(root, &service)
			suite.Nil(err)
			suite.NotNil(service)
			suite.Same(root, service.Context())
			suite.False(service.disposed)
			var service2 *ScopedService
			err = miruken.Resolve(root, &service2)
			suite.Nil(err)
			suite.Same(service, service2)
		})

		suite.Run("SameContextualWithoutQualifier", func() {
			root  := suite.RootContext()
			child := root.NewChild()
			var service *ScopedService
			var childService *ScopedService
			err := miruken.Resolve(root, &service)
			suite.Nil(err)
			suite.Same(root, service.Context())
			err = miruken.Resolve(child, &childService)
			suite.Nil(err)
			suite.Same(service, childService)
		})

		suite.Run("ChildContextAssigned", func() {
			root  := suite.RootContext()
			child := root.NewChild()
			var service *ScopedService
			var childService *ScopedService
			err := miruken.Resolve(root, &service)
			suite.Nil(err)
			suite.NotNil(service)
			err = miruken.Resolve(child, &childService,
				func(c *miruken.ConstraintBuilder) {
					c.WithConstraint(miruken.ScopedQualifier{})
				})
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
			var service *ScopedService
			err := miruken.Resolve(root, &service)
			suite.Nil(err)
			suite.NotNil(service)
			suite.Same(root, service.Context())
			suite.False(service.disposed)
			root.End(nil)
			suite.Nil(service.Context())
			suite.True(service.disposed)
			var service2 *ScopedService
			err = miruken.Resolve(root, &service2)
			suite.NotNil(err)
			suite.Equal("cannot scope instances to an inactive context", err.Error())
			suite.Nil(service2)
		})

		suite.Run("UnmanagedWhenContextNil", func() {
			root := suite.RootContext()
			var service *ScopedService
			err := miruken.Resolve(root, &service)
			suite.Nil(err)
			service.SetContext(nil)
			suite.True(service.disposed)
			var service2 *ScopedService
			err = miruken.Resolve(root, &service2)
			suite.Nil(err)
			suite.NotSame(service, service2)
		})

		suite.Run("RootContextAssigned", func() {
			root   := suite.RootContext()
			child1 := root.NewChild()
			child2 := root.NewChild()
			var service *RootedService
			err := miruken.Resolve(child1, &service)
			suite.Nil(err)
			suite.NotNil(service)
			suite.Same(root, service.Context())
			suite.False(service.disposed)
			var service2 *RootedService
			err = miruken.Resolve(child2, &service2)
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
			var service *ScopedService
			err := miruken.Resolve(root, &service)
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

		suite.Run("Provides", func () {
			suite.Run("Invariant", func() {
				root := suite.RootContext()
				var foo *Foo
				err := miruken.Resolve(root, &foo)
				suite.Nil(err)
				suite.Equal(1, foo.Count())
			})

			suite.Run("Covariant", func() {
				root := suite.RootContextWith(&RootedService{})
				var counter Counter
				err := miruken.Resolve(root, &counter)
				suite.Nil(err)
				suite.Equal(2, counter.Count())
			})
		})

		suite.Run("RejectScopedDependencyInSingleton", func() {
			root := suite.RootContextWith(
				&ScopedService{},
				&LifestyleMismatch{})
			var mismatch *LifestyleMismatch
			err := miruken.Resolve(root, &mismatch)
			suite.Nil(err)
			suite.Nil(mismatch)
		})
	})
}

func TestContextTestSuite(t *testing.T) {
	suite.Run(t, new(ContextTestSuite))
}
