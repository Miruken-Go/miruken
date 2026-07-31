package test

import (
	"context"
	"errors"
	"testing"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/effect"
	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/suite"
)

type (
	// syncEffect completes without any async component.
	syncEffect struct{}

	// asyncValueEffect resolves a promise carrying a distinguishable value.
	asyncValueEffect struct {
		value any
	}

	// erroringEffect fails synchronously.
	erroringEffect struct {
		err error
	}

	// rejectingEffect fails asynchronously.
	rejectingEffect struct {
		err error
	}
)

func (syncEffect) Apply(miruken.HandleContext) (promise.Reflect, error) {
	return nil, nil
}

func (e asyncValueEffect) Apply(miruken.HandleContext) (promise.Reflect, error) {
	return promise.Resolve(e.value), nil
}

func (e erroringEffect) Apply(miruken.HandleContext) (promise.Reflect, error) {
	return nil, e.err
}

func (e rejectingEffect) Apply(miruken.HandleContext) (promise.Reflect, error) {
	return promise.Reject[any](e.err), nil
}

type EffectTestSuite struct {
	suite.Suite
}

func (suite *EffectTestSuite) TestGroup() {
	suite.Run("NoEffects", func() {
		g := effect.Group(context.Background())
		pr, err := g.Apply(miruken.HandleContext{})
		suite.Nil(err)
		suite.Nil(pr)
	})

	suite.Run("AllSynchronous", func() {
		g := effect.Group(context.Background(), syncEffect{}, syncEffect{})
		pr, err := g.Apply(miruken.HandleContext{})
		suite.Nil(err)
		suite.Nil(pr)
	})

	suite.Run("SingleAsync", func() {
		g := effect.Group(context.Background(),
			syncEffect{}, asyncValueEffect{value: "one"})
		pr, err := g.Apply(miruken.HandleContext{})
		suite.Nil(err)
		suite.NotNil(pr)
		v, err := pr.AwaitAny()
		suite.Nil(err)
		suite.Equal("one", v)
	})

	suite.Run("MultipleAsync", func() {
		g := effect.Group(context.Background(),
			asyncValueEffect{value: "one"}, asyncValueEffect{value: "two"})
		pr, err := g.Apply(miruken.HandleContext{})
		suite.Nil(err)
		suite.NotNil(pr)
		v, err := pr.AwaitAny()
		suite.Nil(err)
		suite.ElementsMatch([]any{"one", "two"}, v)
	})

	suite.Run("SynchronousErrorShortCircuits", func() {
		called := false
		failFast := erroringEffect{err: errors.New("boom")}
		neverCalled := asyncValueEffect{value: "unreachable"}
		g := effect.Group(context.Background(), failFast, neverCalled)
		pr, err := g.Apply(miruken.HandleContext{})
		suite.NotNil(err)
		suite.Nil(pr)
		suite.False(called)
	})

	suite.Run("AsyncRejection", func() {
		g := effect.Group(context.Background(),
			rejectingEffect{err: errors.New("failed")})
		pr, err := g.Apply(miruken.HandleContext{})
		suite.Nil(err)
		suite.NotNil(pr)
		_, err = pr.AwaitAny()
		suite.NotNil(err)
		suite.Equal("failed", err.Error())
	})
}

func (suite *EffectTestSuite) TestAsync() {
	suite.Run("WrapsSynchronousEffectAsPromise", func() {
		a, err := effect.Async(context.Background(), syncEffect{})
		suite.Nil(err)
		suite.NotNil(a)
		pr, err := a.Apply(miruken.HandleContext{})
		suite.Nil(err)
		suite.NotNil(pr)
		v, err := pr.AwaitAny()
		suite.Nil(err)
		suite.Equal(struct{}{}, v)
	})

	suite.Run("DiscardsInnerResolvedValue", func() {
		a, err := effect.Async(context.Background(), asyncValueEffect{value: "ignored"})
		suite.Nil(err)
		pr, err := a.Apply(miruken.HandleContext{})
		suite.Nil(err)
		v, err := pr.AwaitAny()
		suite.Nil(err)
		suite.Equal(struct{}{}, v)
	})

	suite.Run("PropagatesInnerRejection", func() {
		a, err := effect.Async(context.Background(),
			rejectingEffect{err: errors.New("failed")})
		suite.Nil(err)
		pr, err := a.Apply(miruken.HandleContext{})
		suite.Nil(err)
		_, err = pr.AwaitAny()
		suite.NotNil(err)
		suite.Equal("failed", err.Error())
	})

	suite.Run("PropagatesSynchronousError", func() {
		a, err := effect.Async(context.Background(),
			erroringEffect{err: errors.New("boom")})
		suite.Nil(err)
		pr, err := a.Apply(miruken.HandleContext{})
		suite.Nil(err)
		_, err = pr.AwaitAny()
		suite.NotNil(err)
		suite.Equal("boom", err.Error())
	})

	suite.Run("NilEffect", func() {
		a, err := effect.Async(context.Background(), nil)
		suite.Nil(err)
		suite.Nil(a)
	})
}

func TestEffectTestSuite(t *testing.T) {
	suite.Run(t, new(EffectTestSuite))
}
