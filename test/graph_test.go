package test

import (
	"iter"
	"testing"

	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
)

type treeNode struct {
	data     any
	parent   miruken.Traversing
	children []miruken.Traversing
}

func (t *treeNode) Parent() miruken.Traversing {
	return t.parent
}

func (t *treeNode) Children() []miruken.Traversing {
	return t.children
}

func (t *treeNode) addChildren(children ...*treeNode) *treeNode {
	for _, child := range children {
		child.parent = t
		t.children = append(t.children, child)
	}
	return t
}

func (t *treeNode) Traverse(
	axis miruken.TraversingAxis,
) iter.Seq[miruken.Traversing] {
	return miruken.TraverseAxis(t, axis)
}

func visit(seq iter.Seq[miruken.Traversing]) []*treeNode {
	var visited []*treeNode
	for node := range seq {
		visited = append(visited, node.(*treeNode))
	}
	return visited
}

type TraversalTestSuite struct {
	suite.Suite
	root,
	child1, child11,
	child2, child21, child22,
	child3, child31, child32, child33 *treeNode
}

func (suite *TraversalTestSuite) SetupTest() {
	suite.root = &treeNode{data: "root"}
	suite.child1 = &treeNode{data: "child1"}
	suite.child11 = &treeNode{data: "child11"}
	suite.child2 = &treeNode{data: "child2"}
	suite.child21 = &treeNode{data: "child21"}
	suite.child22 = &treeNode{data: "child22"}
	suite.child3 = &treeNode{data: "child3"}
	suite.child31 = &treeNode{data: "child31"}
	suite.child32 = &treeNode{data: "child32"}
	suite.child33 = &treeNode{data: "child33"}
	suite.child1.addChildren(suite.child11)
	suite.child2.addChildren(suite.child21, suite.child22)
	suite.child3.addChildren(suite.child31, suite.child32, suite.child33)
	suite.root.addChildren(suite.child1, suite.child2, suite.child3)
}

func (suite *TraversalTestSuite) TestPreOrderTraversal() {
	visited := visit(miruken.TraversePreOrder(suite.root))
	suite.ElementsMatch(visited,
		[]*treeNode{
			suite.root, suite.child1, suite.child11,
			suite.child2, suite.child21, suite.child22,
			suite.child3, suite.child31, suite.child32,
			suite.child33,
		})
}

func (suite *TraversalTestSuite) TestPostOrderTraversal() {
	visited := visit(miruken.TraversePostOrder(suite.root))
	suite.ElementsMatch(visited,
		[]*treeNode{
			suite.child11, suite.child1, suite.child21,
			suite.child22, suite.child2, suite.child31,
			suite.child32, suite.child33, suite.child3,
			suite.root,
		})
}

func (suite *TraversalTestSuite) TestLevelOrderTraversal() {
	visited := visit(miruken.TraverseLevelOrder(suite.root))
	suite.ElementsMatch(visited,
		[]*treeNode{
			suite.root, suite.child1, suite.child2,
			suite.child3, suite.child11, suite.child21,
			suite.child22, suite.child31, suite.child32,
			suite.child33,
		})
}

func (suite *TraversalTestSuite) TestReverseLevelOrderTraversal() {
	visited := visit(miruken.TraverseReverseLevelOrder(suite.root))
	suite.ElementsMatch(visited,
		[]*treeNode{
			suite.child11, suite.child21, suite.child22,
			suite.child31, suite.child32, suite.child33,
			suite.child1, suite.child2, suite.child3,
			suite.root,
		})
}

func (suite *TraversalTestSuite) TestPreOrderStopsEarly() {
	var visited []*treeNode
	for node := range miruken.TraversePreOrder(suite.root) {
		tn := node.(*treeNode)
		visited = append(visited, tn)
		if tn == suite.child1 {
			break
		}
	}
	suite.Equal([]*treeNode{suite.root, suite.child1}, visited)
}

func (suite *TraversalTestSuite) TestCircularityPanics() {
	suite.root.parent = suite.child11
	suite.Panics(func() {
		visit(miruken.TraverseAxis(suite.root, miruken.TraverseAncestor))
	})
}

func TestTraversalTestSuite(t *testing.T) {
	suite.Run(t, new(TraversalTestSuite))
}

type GraphTestSuite struct {
	suite.Suite
}

func (suite *GraphTestSuite) TestTraverseSelf() {
	var root = &treeNode{data: "root"}
	visited := visit(miruken.TraverseAxis(root, miruken.TraverseSelf))
	suite.ElementsMatch(visited, []*treeNode{root})
}

func (suite *GraphTestSuite) TestTraverseRoot() {
	var root = &treeNode{data: "root"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	root.addChildren(child1, child2, child3)
	visited := visit(miruken.TraverseAxis(root, miruken.TraverseRoot))
	suite.ElementsMatch(visited, []*treeNode{root})
}

func (suite *GraphTestSuite) TestTraverseChildren() {
	var root = &treeNode{data: "root"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	child3.addChildren(&treeNode{data: "child31"})
	root.addChildren(child1, child2, child3)
	visited := visit(miruken.TraverseAxis(root, miruken.TraverseChild))
	suite.ElementsMatch(visited, []*treeNode{child1, child2, child3})
}

func (suite *GraphTestSuite) TestTraverseSiblings() {
	var root = &treeNode{data: "root"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	child3.addChildren(&treeNode{data: "child31"})
	root.addChildren(child1, child2, child3)
	visited := visit(miruken.TraverseAxis(child2, miruken.TraverseSibling))
	suite.ElementsMatch(visited, []*treeNode{child1, child3})
}

func (suite *GraphTestSuite) TestTraverseChildrenAndSelf() {
	var root = &treeNode{data: "root"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	child3.addChildren(&treeNode{data: "child31"})
	root.addChildren(child1, child2, child3)
	visited := visit(miruken.TraverseAxis(root, miruken.TraverseSelfOrChild))
	suite.ElementsMatch(visited, []*treeNode{root, child1, child2, child3})
}

func (suite *GraphTestSuite) TestTraverseSiblingAndSelf() {
	var root = &treeNode{data: "root"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	child3.addChildren(&treeNode{data: "child31"})
	root.addChildren(child1, child2, child3)
	visited := visit(miruken.TraverseAxis(child2, miruken.TraverseSelfOrSibling))
	suite.ElementsMatch(visited, []*treeNode{child2, child1, child3})
}

func (suite *GraphTestSuite) TestTraverseAncestors() {
	var root = &treeNode{data: "root"}
	var child = &treeNode{data: "child"}
	var grandChild = &treeNode{data: "grandChild"}
	root.addChildren(child)
	child.addChildren(grandChild)
	visited := visit(miruken.TraverseAxis(grandChild, miruken.TraverseAncestor))
	suite.ElementsMatch(visited, []*treeNode{child, root})
}

func (suite *GraphTestSuite) TestTraverseAncestorsAndSelf() {
	var root = &treeNode{data: "root"}
	var child = &treeNode{data: "child"}
	var grandChild = &treeNode{data: "grandChild"}
	root.addChildren(child)
	child.addChildren(grandChild)
	visited := visit(miruken.TraverseAxis(grandChild, miruken.TraverseSelfOrAncestor))
	suite.ElementsMatch(visited, []*treeNode{grandChild, child, root})
}

func (suite *GraphTestSuite) TestTraverseDescendants() {
	var root = &treeNode{data: "root"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	var child31 = &treeNode{data: "child31"}
	child3.addChildren(child31)
	root.addChildren(child1, child2, child3)
	visited := visit(miruken.TraverseAxis(root, miruken.TraverseDescendant))
	suite.ElementsMatch(visited, []*treeNode{child1, child2, child3, child31})
}

func (suite *GraphTestSuite) TestTraverseDescendantsReverse() {
	var root = &treeNode{data: "root"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	var child31 = &treeNode{data: "child31"}
	child3.addChildren(child31)
	root.addChildren(child1, child2, child3)
	visited := visit(miruken.TraverseAxis(root, miruken.TraverseDescendantReverse))
	suite.ElementsMatch(visited, []*treeNode{child31, child3, child2, child1})
}

func (suite *GraphTestSuite) TestTraverseDescendantsAndSelf() {
	var root = &treeNode{data: "root"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	var child31 = &treeNode{data: "child31"}
	child3.addChildren(child31)
	root.addChildren(child1, child2, child3)
	visited := visit(miruken.TraverseAxis(root, miruken.TraverseSelfOrDescendant))
	suite.ElementsMatch(visited, []*treeNode{root, child1, child2, child3, child31})
}

func (suite *GraphTestSuite) TestTraverseDescendantsAndSelfReverse() {
	var root = &treeNode{data: "root"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	var child31 = &treeNode{data: "child31"}
	child3.addChildren(child31)
	root.addChildren(child1, child2, child3)
	visited := visit(miruken.TraverseAxis(root, miruken.TraverseSelfOrDescendantReverse))
	suite.ElementsMatch(visited, []*treeNode{child31, child1, child2, child3, root})
}

func (suite *GraphTestSuite) TestTraverseAncestorSiblingAndSelf() {
	var root = &treeNode{data: "root"}
	var parent = &treeNode{data: "parent"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	var child31 = &treeNode{data: "child31"}
	child3.addChildren(child31)
	parent.addChildren(child1, child2, child3)
	root.addChildren(parent)
	visited := visit(miruken.TraverseAxis(child3, miruken.TraverseSelfSiblingOrAncestor))
	suite.ElementsMatch(visited, []*treeNode{child3, child1, child2, parent, root})
}

func TestGraphTestSuite(t *testing.T) {
	suite.Run(t, new(GraphTestSuite))
}
