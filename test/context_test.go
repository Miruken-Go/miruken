package test

import (
	"github.com/stretchr/testify/suite"
	"miruken.com/miruken"
	"reflect"
	"testing"
)

type Observer struct {}

func (o *Observer) ContextEnding(
	ctx    *miruken.Context,
	reason  interface{},
) {

}

func (o *Observer) ContextEnded(
	ctx    *miruken.Context,
	reason  interface{},
) {

}

func (o *Observer) ChildContextEnding(
	childCtx *miruken.Context,
	reason    interface{},
) {

}

func (o *Observer) ChildContextEnded(
	childCtx *miruken.Context,
	reason    interface{},
) {

}

type ContextTestSuite struct {
	suite.Suite
	HandleTypes []reflect.Type
}

func (suite *ContextTestSuite) SetupTest() {
	var handleTypes []reflect.Type
	suite.HandleTypes = handleTypes
}

func (suite *ContextTestSuite) InferenceRoot() miruken.Handler {
	return miruken.NewRootHandler(miruken.WithHandlerTypes(suite.HandleTypes...))
}

func (suite *ContextTestSuite) InferenceRootWith(
	handlerTypes ... reflect.Type) miruken.Handler {
	return miruken.NewRootHandler(miruken.WithHandlerTypes(handlerTypes...))
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
		data := &Observer{}
		context := miruken.NewContext()
		context.Store(data)
		var resolve *Observer
		err := miruken.Resolve(context, &resolve)
		suite.Nil(err)
		suite.Same(data, resolve)
	})

	suite.Run("TraverseAncestorsByDefault", func () {
		data       := &Observer{}
		root       := miruken.NewContext()
		child      := root.NewChild()
		grandChild := child.NewChild()
		root.Store(data)
		var resolve *Observer
		err := miruken.Resolve(grandChild, &resolve)
		suite.Nil(err)
		suite.Same(data, resolve)
	})

	suite.Run("TraverseSelf", func () {
		data       := &Observer{}
		root       := miruken.NewContext()
		child      := root.NewChild()
		root.Store(data)
		var resolve *Observer
		err := miruken.Resolve(miruken.Build(child, miruken.WithSelf), &resolve)
		suite.Nil(err)
		suite.Nil(resolve)
		err = miruken.Resolve(miruken.Build(root, miruken.WithSelf), &resolve)
		suite.Nil(err)
		suite.Same(data, resolve)
	})

	suite.Run("TraverseRoot", func () {
		data       := &Observer{}
		root       := miruken.NewContext()
		child      := root.NewChild()
		child.Store(data)
		var resolve *Observer
		err := miruken.Resolve(miruken.Build(child, miruken.WithRoot), &resolve)
		suite.Nil(err)
		suite.Nil(resolve)
		root.Store(data)
		err = miruken.Resolve(miruken.Build(root, miruken.WithRoot), &resolve)
		suite.Nil(err)
		suite.Same(data, resolve)
	})

	suite.Run("TraverseChildren", func () {
		data       := &Observer{}
		root       := miruken.NewContext()
		root.NewChild()
		child2     := root.NewChild()
		child3     := root.NewChild()
		grandChild := child3.NewChild()
		child2.Store(data)
		var resolve *Observer
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

	suite.Run("TraverseSiblings", func () {
		data       := &Observer{}
		root       := miruken.NewContext()
		root.NewChild()
		child2     := root.NewChild()
		child3     := root.NewChild()
		grandChild := child3.NewChild()
		child3.Store(data)
		var resolve *Observer
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

	suite.Run("TraverseChildrenOrSelf", func () {
		data       := &Observer{}
		root       := miruken.NewContext()
		child1     := root.NewChild()
		root.NewChild()
		child3     := root.NewChild()
		grandChild := child3.NewChild()
		child3.Store(data)
		var resolve *Observer
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

	suite.Run("TraverseSiblingsOrSelf", func () {
		data       := &Observer{}
		root       := miruken.NewContext()
		root.NewChild()
		child2     := root.NewChild()
		child3     := root.NewChild()
		grandChild := child3.NewChild()
		child3.Store(data)
		var resolve *Observer
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

	suite.Run("TraverseAncestors", func () {
		data       := &Observer{}
		root       := miruken.NewContext()
		child      := root.NewChild()
		grandChild  := child.NewChild()
		root.Store(data)
		var resolve *Observer
		err := miruken.Resolve(miruken.Build(root, miruken.WithAncestor), &resolve)
		suite.Nil(err)
		suite.Nil(resolve)
		err = miruken.Resolve(miruken.Build(grandChild, miruken.WithAncestor), &resolve)
		suite.Nil(err)
		suite.Same(data, resolve)
	})

	suite.Run("TraverseAncestorsOrSelf", func () {
		data       := &Observer{}
		root       := miruken.NewContext()
		child      := root.NewChild()
		grandChild  := child.NewChild()
		root.Store(data)
		var resolve *Observer
		err := miruken.Resolve(miruken.Build(root, miruken.WithSelfOrAncestor), &resolve)
		suite.Nil(err)
		suite.Same(data, resolve)
		err = miruken.Resolve(miruken.Build(grandChild, miruken.WithSelfOrAncestor), &resolve)
		suite.Nil(err)
		suite.Same(data, resolve)
	})

	suite.Run("TraverseDescendants", func () {
		data       := &Observer{}
		root       := miruken.NewContext()
		root.NewChild()
		child2     := root.NewChild()
		child3     := root.NewChild()
		grandChild := child3.NewChild()
		grandChild.Store(data)
		var resolve *Observer
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

	suite.Run("TraverseDescendantsOrSelf", func () {
		data       := &Observer{}
		root       := miruken.NewContext()
		root.NewChild()
		child2     := root.NewChild()
		child3     := root.NewChild()
		grandChild := child3.NewChild()
		grandChild.Store(data)
		var resolve *Observer
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

	suite.Run("TraverseDescendantsOrSELF", func () {
		data       := &Observer{}
		root       := miruken.NewContext()
		root.NewChild()
		child2     := root.NewChild()
		child3     := root.NewChild()
		child3.NewChild()
		root.Store(data)
		var resolve *Observer
		err := miruken.Resolve(miruken.Build(child2, miruken.WithSelfOrDescendant), &resolve)
		suite.Nil(err)
		suite.Nil(resolve)
		err = miruken.Resolve(miruken.Build(root, miruken.WithSelfOrDescendant), &resolve)
		suite.Nil(err)
		suite.Same(data, resolve)
	})

	suite.Run("TraverseAncestorsSIBLINGSorSelf", func () {
		data       := &Observer{}
		root       := miruken.NewContext()
		root.NewChild()
		child2     := root.NewChild()
		child3     := root.NewChild()
		grandChild := child3.NewChild()
		child2.Store(data)
		var resolve *Observer
		err := miruken.Resolve(miruken.Build(grandChild, miruken.WithSelfSiblingOrAncestor), &resolve)
		suite.Nil(err)
		suite.Nil(resolve)
		err = miruken.Resolve(miruken.Build(child3, miruken.WithSelfSiblingOrAncestor), &resolve)
		suite.Nil(err)
		suite.Same(data, resolve)
	})

	suite.Run("TraverseANCESTORSSiblingsOrSelf", func () {
		data       := &Observer{}
		root       := miruken.NewContext()
		root.NewChild()
		root.NewChild()
		child3     := root.NewChild()
		grandChild := child3.NewChild()
		child3.Store(data)
		var resolve *Observer
		err := miruken.Resolve(miruken.Build(grandChild, miruken.WithSelfSiblingOrAncestor), &resolve)
		suite.Nil(err)
		suite.Same(data, resolve)
	})
}

type ContextualService struct {
	miruken.ContextualBase
}

func (c *ContextualService) SetContext(ctx *miruken.Context) {
	c.ChangeContext(c, ctx)
}

func (suite *ContextTestSuite) TestContextual() {
	suite.Run("ContextInitiallyEmpty", func () {
		service := ContextualService{}
		suite.Nil(service.Context())
	})

	suite.Run("SetContext", func () {
		service := ContextualService{}
		root    := miruken.NewContext()
		service.SetContext(root)
		suite.Same(root, service.Context())
	})

	suite.Run("AddsContextualToContext", func () {
		service := ContextualService{}
		root    := miruken.NewContext()
		service.SetContext(root)
		var services []*ContextualService
		if err := miruken.ResolveAll(root, &services); err == nil {
			suite.NotNil(services)
			suite.Len(services, 1)
			suite.Same(&service, services[0])
		} else {
			suite.Failf("unexpected error", err.Error())
		}
	})
}

func TestContextTestSuite(t *testing.T) {
	suite.Run(t, new(ContextTestSuite))
}
