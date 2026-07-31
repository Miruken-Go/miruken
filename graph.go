package miruken

import (
	"container/list"
	"fmt"
	"iter"

	"github.com/miruken-go/miruken/internal"
)

// TraversingAxis defines a path of traversal.
type TraversingAxis uint

const (
	TraverseSelf = TraversingAxis(iota)
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

// Traversing represents a node in a graph.
type Traversing interface {
	Parent() Traversing
	Children() []Traversing
	Traverse(axis TraversingAxis) iter.Seq[Traversing]
}

// TraverseAxis traverses a node over an axis.
// Panics with TraversalCircularityError if a cycle is detected -
// Traversing implementations are expected to form a tree.
func TraverseAxis(
	node Traversing,
	axis TraversingAxis,
) iter.Seq[Traversing] {
	return func(yield func(Traversing) bool) {
		switch axis {
		case TraverseSelf:
			traverseSelf(node, yield)
		case TraverseRoot:
			traverseRoot(node, yield)
		case TraverseChild:
			traverseChildren(node, yield, false)
		case TraverseSibling:
			traverseSelfSiblingOrAncestor(node, yield, false, false)
		case TraverseSelfOrChild:
			traverseChildren(node, yield, true)
		case TraverseSelfOrSibling:
			traverseSelfSiblingOrAncestor(node, yield, true, false)
		case TraverseAncestor:
			traverseAncestors(node, yield, false)
		case TraverseSelfOrAncestor:
			traverseAncestors(node, yield, true)
		case TraverseDescendant:
			traverseDescendants(node, yield, false)
		case TraverseDescendantReverse:
			traverseDescendantsReverse(node, yield, false)
		case TraverseSelfOrDescendant:
			traverseDescendants(node, yield, true)
		case TraverseSelfOrDescendantReverse:
			traverseDescendantsReverse(node, yield, true)
		case TraverseSelfSiblingOrAncestor:
			traverseSelfSiblingOrAncestor(node, yield, true, true)
		default:
			panic(fmt.Sprintf("unrecognized axis %v", axis))
		}
	}
}

func traverseSelf(
	node  Traversing,
	yield func(Traversing) bool,
) {
	yield(node)
}

func traverseRoot(
	node  Traversing,
	yield func(Traversing) bool,
) {
	root := node
	visited := make(traversalHistory)
	for parent := root.Parent(); !internal.IsNil(parent); parent = parent.Parent() {
		checkTraversalCircularity(parent, visited)
		root = parent
	}
	yield(root)
}

func traverseChildren(
	node     Traversing,
	yield    func(Traversing) bool,
	withSelf bool,
) {
	if withSelf && !yield(node) {
		return
	}
	for _, child := range node.Children() {
		if !yield(child) {
			return
		}
	}
}

func traverseAncestors(
	node     Traversing,
	yield    func(Traversing) bool,
	withSelf bool,
) {
	if withSelf && !yield(node) {
		return
	}
	parent := node.Parent()
	visited := make(traversalHistory)
	for !internal.IsNil(parent) {
		checkTraversalCircularity(parent, visited)
		if !yield(parent) {
			return
		}
		parent = parent.Parent()
	}
}

func traverseDescendants(
	node     Traversing,
	yield    func(Traversing) bool,
	withSelf bool,
) {
	for child := range TraverseLevelOrder(node) {
		if child != node || withSelf {
			if !yield(child) {
				return
			}
		}
	}
}

func traverseDescendantsReverse(
	node     Traversing,
	yield    func(Traversing) bool,
	withSelf bool,
) {
	for child := range TraverseReverseLevelOrder(node) {
		if child != node || withSelf {
			if !yield(child) {
				return
			}
		}
	}
}

func traverseSelfSiblingOrAncestor(
	node          Traversing,
	yield         func(Traversing) bool,
	withSelf      bool,
	withAncestors bool,
) {
	if withSelf && !yield(node) {
		return
	}
	parent := node.Parent()
	if internal.IsNil(parent) {
		return
	}
	for _, sibling := range parent.Children() {
		if sibling == node {
			continue
		}
		if !yield(sibling) {
			return
		}
	}
	if withAncestors {
		traverseAncestors(parent, yield, true)
	}
}

// TraversalCircularityError reports a traversal circularity.
type TraversalCircularityError struct {
	culprit Traversing
}

func (e TraversalCircularityError) Culprit() Traversing {
	return e.culprit
}

func (e TraversalCircularityError) Error() string {
	return fmt.Sprintf("circularity detected for node %v", e.culprit)
}

type traversalHistory map[Traversing]bool

// TraversePreOrder traverses the node using the pre-order algorithm.
// Panics with TraversalCircularityError if a cycle is detected.
func TraversePreOrder(node Traversing) iter.Seq[Traversing] {
	return func(yield func(Traversing) bool) {
		traversePreOrder(node, yield, make(traversalHistory))
	}
}

func traversePreOrder(
	node    Traversing,
	yield   func(Traversing) bool,
	visited traversalHistory,
) bool {
	if node == nil {
		return true
	}
	checkTraversalCircularity(node, visited)
	if !yield(node) {
		return false
	}
	for _, child := range node.Children() {
		if !traversePreOrder(child, yield, visited) {
			return false
		}
	}
	return true
}

// TraversePostOrder traverses the node using the post-order algorithm.
// Panics with TraversalCircularityError if a cycle is detected.
func TraversePostOrder(node Traversing) iter.Seq[Traversing] {
	return func(yield func(Traversing) bool) {
		traversePostOrder(node, yield, make(traversalHistory))
	}
}

func traversePostOrder(
	node    Traversing,
	yield   func(Traversing) bool,
	history traversalHistory,
) bool {
	if node == nil {
		return true
	}
	checkTraversalCircularity(node, history)
	for _, child := range node.Children() {
		if !traversePostOrder(child, yield, history) {
			return false
		}
	}
	return yield(node)
}

// TraverseLevelOrder traverses the node using the level-order algorithm.
// Panics with TraversalCircularityError if a cycle is detected.
func TraverseLevelOrder(node Traversing) iter.Seq[Traversing] {
	return func(yield func(Traversing) bool) {
		if node == nil {
			return
		}
		history := make(traversalHistory)
		queue := list.New()
		queue.PushBack(node)
		for queue.Len() > 0 {
			front := queue.Front()
			queue.Remove(front)
			next := front.Value.(Traversing)
			checkTraversalCircularity(next, history)
			if !yield(next) {
				return
			}
			for _, child := range next.Children() {
				if !internal.IsNil(child) {
					queue.PushBack(child)
				}
			}
		}
	}
}

// TraverseReverseLevelOrder traverses the node using the reverse level-order algorithm.
// Panics with TraversalCircularityError if a cycle is detected.
func TraverseReverseLevelOrder(node Traversing) iter.Seq[Traversing] {
	return func(yield func(Traversing) bool) {
		if node == nil {
			return
		}
		history := make(traversalHistory)
		queue := list.New()
		queue.PushBack(node)
		stack := list.New()
		for queue.Len() > 0 {
			front := queue.Front()
			queue.Remove(front)
			next := front.Value.(Traversing)
			checkTraversalCircularity(next, history)
			stack.PushBack(next)
			level := list.New()
			for _, child := range next.Children() {
				if !internal.IsNil(child) {
					level.PushFront(child)
				}
			}
			for e := level.Front(); e != nil; e = e.Next() {
				queue.PushBack(e.Value)
			}
		}
		for stack.Len() > 0 {
			back := stack.Back()
			stack.Remove(back)
			next := back.Value.(Traversing)
			if !yield(next) {
				return
			}
		}
	}
}

func checkTraversalCircularity(
	node    Traversing,
	history traversalHistory,
) {
	if _, ok := history[node]; ok {
		panic(TraversalCircularityError{node})
	}
	history[node] = true
}
