package miruken

import (
	"container/list"
	"fmt"
)

// TraversingAxis defines a path of traversal.
type TraversingAxis uint

const (
	TraverseSelf TraversingAxis = iota
	TraverseRoot
	TraverseChild
	TraverseSibling
	TraverseAncestor
	TraverseDescendant
	TraverseDescendantReverse
	TraverseSelfOrChild
	TraverseSelfOrSibling
	TraverseSelfOrAncestor
	TraverseSelfOrDescendant
	TraverseSelfOrDescendantReverse
	TraverseSelfSiblingOrAncestor
)

// TraversalVisitor is called during traversal.
type TraversalVisitor interface {
	VisitTraversal(node Traversing) (stop bool, err error)
}

type TraversalVisitorFunc func(node Traversing) (stop bool, err error)

func (f TraversalVisitorFunc) VisitTraversal(node Traversing) (bool, error) {
	return f(node)
}

// Traversing represents a node in a graph.
type Traversing interface {
	Parent() Traversing
	Children() []Traversing
	Traverse(axis TraversingAxis, visitor TraversalVisitor) error
}

// TraverseAxis traverses a node over an axis.
func TraverseAxis(
	node    Traversing,
	axis    TraversingAxis,
	visitor TraversalVisitor,
) error {
	if visitor == nil {
		return nil
	}
	switch axis {
	case TraverseSelf:
		return traverseSelf(node, visitor)
	case TraverseRoot:
		return traverseRoot(node, visitor)
	case TraverseChild:
		return traverseChildren(node, visitor, false)
	case TraverseSibling:
		return traverseSelfSiblingOrAncestor(node, visitor, false, false)
	case TraverseSelfOrChild:
		return traverseChildren(node, visitor, true)
	case TraverseSelfOrSibling:
		return traverseSelfSiblingOrAncestor(node, visitor, true, false)
	case TraverseAncestor:
		return traverseAncestors(node, visitor, false)
	case TraverseSelfOrAncestor:
		return traverseAncestors(node, visitor, true)
	case TraverseDescendant:
		return traverseDescendants(node, visitor, false)
	case TraverseDescendantReverse:
		return traverseDescendantsReverse(node, visitor, false)
	case TraverseSelfOrDescendant:
		return traverseDescendants(node, visitor, true)
	case TraverseSelfOrDescendantReverse:
		return traverseDescendantsReverse(node, visitor, true)
	case TraverseSelfSiblingOrAncestor:
		return traverseSelfSiblingOrAncestor(node, visitor, true, true)
	}
	panic(fmt.Sprintf("unrecognized axis %v", axis))
}

func traverseSelf(
	node    Traversing,
	visitor TraversalVisitor,
) error {
	_, err := visitor.VisitTraversal(node)
	return err
}

func traverseRoot(
	node    Traversing,
	visitor TraversalVisitor,
) error {
	root    := node
	visited := make(traversalHistory)
	for parent := root.Parent(); parent != nil; {
		if err := checkTraversalCircularity(parent, visited); err != nil {
			return err
		}
		root = parent
	}
	_, err := visitor.VisitTraversal(root)
	return err
}

func traverseChildren(
	node    Traversing,
	visitor TraversalVisitor,
	withSelf bool,
) error {
	if withSelf {
		if _, err := visitor.VisitTraversal(node); err != nil {
			return err
		}
	}
	for _, child := range node.Children() {
		if _, err := visitor.VisitTraversal(child); err != nil {
			return err
		}
	}
	return nil
}

func traverseAncestors(
	node     Traversing,
	visitor  TraversalVisitor,
	withSelf bool,
) error {
	if withSelf {
		if _, err := visitor.VisitTraversal(node); err != nil {
			return err
		}
	}
	parent  := node.Parent()
	visited := make(traversalHistory)
	for parent != nil {
		if err := checkTraversalCircularity(parent, visited); err != nil {
			return err
		}
		if _, err := visitor.VisitTraversal(parent); err != nil {
			return err
		}
		parent = parent.Parent()
	}
	return nil
}

func traverseDescendants(
	node     Traversing,
	visitor  TraversalVisitor,
	withSelf bool,
) error {
	return TraverseLevelOrder(node, TraversalVisitorFunc(
		func(child Traversing) (bool, error) {
			if child != node || withSelf {
				return visitor.VisitTraversal(child)
			}
			return false, nil
		}))
}

func traverseDescendantsReverse(
	node     Traversing,
	visitor  TraversalVisitor,
	withSelf bool,
) error {
	return TraverseReverseLevelOrder(node, TraversalVisitorFunc(
		func(child Traversing) (bool, error) {
			if child != node || withSelf {
				return visitor.VisitTraversal(child)
			}
			return false, nil
		}))
}

func traverseSelfSiblingOrAncestor(
	node          Traversing,
	visitor       TraversalVisitor,
	withSelf      bool,
	withAncestors bool,
) error {
	if withSelf {
		if _, err := visitor.VisitTraversal(node); err != nil {
			return err
		}
	}
	parent := node.Parent()
	if parent == nil {
		return nil
	}
	for _, sibling := range parent.Children() {
		if sibling == node {
			continue
		}
		if stop, err := visitor.VisitTraversal(sibling); stop || err != nil {
			return err
		}
	}
	if withAncestors {
		return traverseAncestors(parent, visitor, true)
	}
	return nil
}

// TraversalCircularityError reports a traversal circularity.
type TraversalCircularityError struct {
	culprit Traversing
}

func (e TraversalCircularityError) Culprit() Traversing {
	return e.culprit
}

func (e TraversalCircularityError) Error() string {
	return fmt.Sprintf("circularity detected for node %#v", e.culprit)
}

type traversalHistory map[Traversing]bool

// TraversePreOrder traverse the node using pre-order algorithm.
func TraversePreOrder(
	node    Traversing,
	visitor TraversalVisitor,
) error {
	_, err := traversePreOrder(node, visitor, make(traversalHistory))
	return err
}

func traversePreOrder(
	node    Traversing,
	visitor TraversalVisitor,
	visited traversalHistory,
) (stop bool, err error) {
	if node == nil || visitor == nil {
		return true, nil
	}
	if err = checkTraversalCircularity(node, visited); err != nil {
		return true, err
	}
	if stop, err = visitor.VisitTraversal(node); stop || err != nil {
		return stop, err
	}
	err = TraverseAxis(node, TraverseChild, TraversalVisitorFunc(
		func(child Traversing) (bool, error) {
			return traversePreOrder(child, visitor, visited)
		}))
	return false, err
}

// TraversePostOrder traverse the node using post-order algorithm.
func TraversePostOrder(
	node    Traversing,
	visitor TraversalVisitor,
) error {
	_, err := traversePostOrder(node, visitor, make(traversalHistory))
	return err
}

func traversePostOrder(
	node    Traversing,
	visitor TraversalVisitor,
	visited traversalHistory,
) (stop bool, err error) {
	if node == nil || visitor == nil {
		return true, nil
	}
	if err = checkTraversalCircularity(node, visited); err != nil {
		return true, err
	}
	if err = TraverseAxis(node, TraverseChild, TraversalVisitorFunc(
		func(child Traversing) (bool, error) {
			return traversePostOrder(child, visitor, visited)
		})); err != nil {
		return false, err
	}
	return visitor.VisitTraversal(node)
}

// TraverseLevelOrder traverse the node using level-order algorithm.
func TraverseLevelOrder(
	node    Traversing,
	visitor TraversalVisitor,
) error {
	_, err := traverseLevelOrder(node, visitor, make(traversalHistory))
	return err
}

func traverseLevelOrder(
	node    Traversing,
	visitor TraversalVisitor,
	visited traversalHistory,
) (stop bool, err error) {
	if node == nil || visitor == nil {
		return true, nil
	}
	queue := list.New()
	queue.PushBack(node)
	for queue.Len() > 0 {
		front := queue.Front()
		queue.Remove(front)
		next := front.Value.(Traversing)
		if err = checkTraversalCircularity(next, visited); err != nil {
			return true, err
		}
		if stop, err := visitor.VisitTraversal(next); stop || err != nil {
			return stop, err
		}
		if err = TraverseAxis(next, TraverseChild, TraversalVisitorFunc(
			func(child Traversing) (bool, error) {
				if child != nil {
					queue.PushBack(child)
				}
				return false, nil
			})); err != nil {
			return false, err
		}
	}
	return false, nil
}

// TraverseReverseLevelOrder traverse the node using reverse level-order algorithm.
func TraverseReverseLevelOrder(
	node    Traversing,
	visitor TraversalVisitor,
) error {
	_, err := traverseReverseLevelOrder(node, visitor, make(traversalHistory))
	return err
}

func traverseReverseLevelOrder(
	node    Traversing,
	visitor TraversalVisitor,
	visited traversalHistory,
) (stop bool, err error) {
	if node == nil || visitor == nil {
		return true, nil
	}
	queue := list.New()
	queue.PushBack(node)
	stack := list.New()
	for queue.Len() > 0 {
		front := queue.Front()
		queue.Remove(front)
		next := front.Value.(Traversing)
		if err = checkTraversalCircularity(next, visited); err != nil {
			return true, err
		}
		stack.PushBack(next)
		level := list.New()
		if err = TraverseAxis(next, TraverseChild, TraversalVisitorFunc(
			func(child Traversing) (bool, error) {
				if child != nil {
					level.PushFront(child)
				}
				return false, nil
			})); err != nil {
			return false, err
		}
		for e := level.Front(); e != nil; e = e.Next() {
			queue.PushBack(e.Value)
		}
	}
	for stack.Len() > 0 {
		back := stack.Back()
		stack.Remove(back)
		next := back.Value.(Traversing)
		if stop, err := visitor.VisitTraversal(next); stop || err != nil {
			return stop, err
		}
	}
	return false, nil
}

func checkTraversalCircularity(
	node    Traversing,
	visited traversalHistory,
) error {
	if _, ok := visited[node]; ok {
		return TraversalCircularityError{node}
	}
	visited[node] = true
	return nil
}