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

func Axis(axis TraversingAxis) BuilderFunc {
	return func(handler Handler) Handler {
		if axisHandler, ok := handler.(HandlerAxis); ok {
			return &axisScope{axisHandler, axis}
		}
		return handler
	}
}

var SelfAxis                    = Axis(TraverseSelf)
var RootAxis                    = Axis(TraverseRoot)
var ChildAxis                   = Axis(TraverseChild)
var SiblingAxis                 = Axis(TraverseSibling)
var AncestorAxis                = Axis(TraverseAncestor)
var DescendantAxis              = Axis(TraverseDescendant)
var DescendantReverseAxis       = Axis(TraverseDescendantReverse)
var SelfOrChildAxis             = Axis(TraverseSelfOrChild)
var SelfOrSiblingAxis           = Axis(TraverseSelfOrSibling)
var SelfOrAncestorAxis          = Axis(TraverseSelfOrAncestor)
var SelfOrDescendantAxis        = Axis(TraverseSelfOrDescendant)
var SelfOrDescendantReverseAxis = Axis(TraverseSelfOrDescendant)
var SelfSiblingOrAncestorAxis   = Axis(TraverseSelfSiblingOrAncestor)

var Publish = ComposeBuilders(SelfOrDescendantAxis, Notify)


