package miruken

type (
	// HandlerAxis extends Handler with traversal.
	HandlerAxis interface {
		Handler
		HandleAxis(
			axis     TraversingAxis,
			callback any,
			greedy   bool,
			composer Handler,
		) HandleResult
	}

	// axisScope applies axis traversal to a Handler.
 	axisScope struct {
		HandlerAxis
		axis TraversingAxis
	}
)

func (a *axisScope) Handle(
	callback any,
	greedy   bool,
	composer Handler,
) HandleResult {
	if composer == nil {
		composer = &compositionScope{a}
	}
	if _, ok := callback.(*Composition); ok {
		return a.HandlerAxis.Handle(callback, greedy, composer)
	}
	return a.HandlerAxis.HandleAxis(a.axis, callback, greedy, composer)
}

func WithAxis(axis TraversingAxis) Builder {
	return BuilderFunc(func(handler Handler) Handler {
		if axisHandler, ok := handler.(HandlerAxis); ok {
			return &axisScope{axisHandler, axis}
		}
		return handler
	})
}

var WithSelf                    = WithAxis(TraverseSelf)
var WithRoot                    = WithAxis(TraverseRoot)
var WithChild                   = WithAxis(TraverseChild)
var WithSibling                 = WithAxis(TraverseSibling)
var WithAncestor                = WithAxis(TraverseAncestor)
var WithDescendant              = WithAxis(TraverseDescendant)
var WithDescendantReverse       = WithAxis(TraverseDescendantReverse)
var WithSelfOrChild             = WithAxis(TraverseSelfOrChild)
var WithSelfOrSibling           = WithAxis(TraverseSelfOrSibling)
var WithSelfOrAncestor          = WithAxis(TraverseSelfOrAncestor)
var WithSelfOrDescendant        = WithAxis(TraverseSelfOrDescendant)
var WithSelfOrDescendantReverse = WithAxis(TraverseSelfOrDescendant)
var WithSelfSiblingOrAncestor   = WithAxis(TraverseSelfSiblingOrAncestor)

var WithPublish = ComposeBuilders(WithSelfOrDescendant, WithNotify)


