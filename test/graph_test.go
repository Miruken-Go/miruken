package test

import (
	"github.com/stretchr/testify/suite"
	"miruken.com/miruken"
	"testing"
)

type treeNode struct {
	data interface{}
	parent miruken.Traversing
	children []miruken.Traversing
}

func (t *treeNode) Parent() miruken.Traversing {
	return t.parent
}

func (t *treeNode) Children() []miruken.Traversing {
	return t.children
}

func (t *treeNode) addChildren(children ... *treeNode) *treeNode{
	for _, child := range children {
		child.parent = t
		t.children = append(t.children, child)
	}
	return t
}

func (t *treeNode) Traverse(
	axis    miruken.TraversingAxis,
	visitor miruken.TraversalVisitor,
) error {
	return miruken.TraverseAxis(t, axis, visitor)
}

type TraversalTestSuite struct {
	suite.Suite
	visited []*treeNode
	root,
	child1, child11,
	child2, child21, child22,
	child3, child31, child32, child33 *treeNode
}

func (suite *TraversalTestSuite) SetupTest() {
	suite.visited = make([]*treeNode, 0)
	suite.root    = &treeNode{data: "root"}
	suite.child1  = &treeNode{data: "child1"}
	suite.child11 = &treeNode{data: "child11"}
	suite.child2  = &treeNode{data: "child2"}
	suite.child21 = &treeNode{data: "child21"}
	suite.child22 = &treeNode{data: "child22"}
	suite.child3  = &treeNode{data: "child3"}
	suite.child31 = &treeNode{data: "child31"}
	suite.child32 = &treeNode{data: "child32"}
	suite.child33 = &treeNode{data: "child33"}
	suite.child1.addChildren(suite.child11)
	suite.child2.addChildren(suite.child21, suite.child22)
	suite.child3.addChildren(suite.child31, suite.child32, suite.child33)
	suite.root.addChildren(suite.child1, suite.child2, suite.child3)
}

func (suite *TraversalTestSuite) VisitTraversal(
	node miruken.Traversing,
) (stop bool, err error) {
	suite.visited = append(suite.visited, node.(*treeNode))
	return false, nil
}

func (suite *TraversalTestSuite) Visited(expected ... *treeNode) {
	suite.ElementsMatch(suite.visited, expected)
}

func (suite *TraversalTestSuite) TestPreOrderTraversal() {
	err := miruken.TraversePreOrder(suite.root, suite)
	suite.Nil(err)
	suite.Visited(
		suite.root,   suite.child1,  suite.child11,
		suite.child2, suite.child21, suite.child22,
		suite.child3, suite.child31, suite.child32,
		suite.child33)
}

func (suite *TraversalTestSuite) TestPostOrderTraversal() {
	err := miruken.TraversePostOrder(suite.root, suite)
	suite.Nil(err)
	suite.Visited(
		suite.child11, suite.child1,  suite.child21,
		suite.child22, suite.child2,  suite.child31,
		suite.child32, suite.child33, suite.child3,
		suite.root)
}

func (suite *TraversalTestSuite) TestLevelOrderTraversal() {
	err := miruken.TraverseLevelOrder(suite.root, suite)
	suite.Nil(err)
	suite.Visited(
		suite.root,    suite.child1,  suite.child2,
		suite.child3,  suite.child11, suite.child21,
		suite.child22, suite.child31, suite.child32,
		suite.child33)
}

func (suite *TraversalTestSuite) TestReverseLevelOrderTraversal() {
	err := miruken.TraverseReverseLevelOrder(suite.root, suite)
	suite.Nil(err)
	suite.Visited(
		suite.child11, suite.child21, suite.child22,
		suite.child31, suite.child32, suite.child33,
		suite.child1,  suite.child2,  suite.child3,
		suite.root)
}

func TestTraversalTestSuite(t *testing.T) {
	suite.Run(t, new(TraversalTestSuite))
}

type TraversingTestSuite struct {
	suite.Suite
	visited []*treeNode
}

func (suite *TraversingTestSuite) SetupTest() {
	suite.visited = make([]*treeNode, 0)
}

func (suite *TraversingTestSuite) VisitTraversal(
	node miruken.Traversing,
) (stop bool, err error) {
	suite.visited = append(suite.visited, node.(*treeNode))
	return false, nil
}

func (suite *TraversingTestSuite) Visited(expected ... *treeNode) {
	suite.ElementsMatch(suite.visited, expected)
}

func (suite *TraversingTestSuite) TestTraverseSelf() {
	var root = &treeNode{data: "root"}
	err := miruken.TraverseAxis(root, miruken.TraverseSelf, suite)
	suite.Nil(err)
	suite.Visited(root)
}

func (suite *TraversingTestSuite) TestTraverseRoot() {
	var root   = &treeNode{data: "root"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	root.addChildren(child1, child2, child3)
	err := miruken.TraverseAxis(root, miruken.TraverseRoot, suite)
	suite.Nil(err)
	suite.Visited(root)
}

func (suite *TraversingTestSuite) TestTraverseChildren() {
	var root   = &treeNode{data: "root"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	child3.addChildren(&treeNode{data: "child31"})
	root.addChildren(child1, child2, child3)
	err := miruken.TraverseAxis(root, miruken.TraverseChild, suite)
	suite.Nil(err)
	suite.Visited(child1, child2, child3)
}

func (suite *TraversingTestSuite) TestTraverseSiblings() {
	var root   = &treeNode{data: "root"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	child3.addChildren(&treeNode{data: "child31"})
	root.addChildren(child1, child2, child3)
	err := miruken.TraverseAxis(child2, miruken.TraverseSibling, suite)
	suite.Nil(err)
	suite.Visited(child1, child3)
}

func (suite *TraversingTestSuite) TestTraverseChildrenAndSelf() {
	var root   = &treeNode{data: "root"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	child3.addChildren(&treeNode{data: "child31"})
	root.addChildren(child1, child2, child3)
	err := miruken.TraverseAxis(root, miruken.TraverseSelfOrChild, suite)
	suite.Nil(err)
	suite.Visited(root, child1, child2, child3)
}

func (suite *TraversingTestSuite) TestTraverseSiblingAndSelf() {
	var root   = &treeNode{data: "root"}
	var child1 = &treeNode{data: "child1"}
	var child2 = &treeNode{data: "child2"}
	var child3 = &treeNode{data: "child3"}
	child3.addChildren(&treeNode{data: "child31"})
	root.addChildren(child1, child2, child3)
	err := miruken.TraverseAxis(child2, miruken.TraverseSelfOrSibling, suite)
	suite.Nil(err)
	suite.Visited(child2, child1, child3)
}

func (suite *TraversingTestSuite) TestTraverseAncestors() {
	var root       = &treeNode{data: "root"}
	var child      = &treeNode{data: "child"}
	var grandChild = &treeNode{data: "grandChild"}
	root.addChildren(child)
	child.addChildren(grandChild)
	err := miruken.TraverseAxis(grandChild, miruken.TraverseAncestor, suite)
	suite.Nil(err)
	suite.Visited(child, root)
}

func (suite *TraversingTestSuite) TestTraverseAncestorsAndSelf() {
	var root       = &treeNode{data: "root"}
	var child      = &treeNode{data: "child"}
	var grandChild = &treeNode{data: "grandChild"}
	root.addChildren(child)
	child.addChildren(grandChild)
	err := miruken.TraverseAxis(grandChild, miruken.TraverseSelfOrAncestor, suite)
	suite.Nil(err)
	suite.Visited(grandChild, child, root)
}

func (suite *TraversingTestSuite) TestTraverseDescendants() {
	var root    = &treeNode{data: "root"}
	var child1  = &treeNode{data: "child1"}
	var child2  = &treeNode{data: "child2"}
	var child3  = &treeNode{data: "child3"}
	var child31 = &treeNode{data: "child31"}
	child3.addChildren(child31)
	root.addChildren(child1, child2, child3)
	err := miruken.TraverseAxis(root, miruken.TraverseDescendant, suite)
	suite.Nil(err)
	suite.Visited(child1, child2, child3, child31)
}

func (suite *TraversingTestSuite) TestTraverseDescendantsReverse() {
	var root    = &treeNode{data: "root"}
	var child1  = &treeNode{data: "child1"}
	var child2  = &treeNode{data: "child2"}
	var child3  = &treeNode{data: "child3"}
	var child31 = &treeNode{data: "child31"}
	child3.addChildren(child31)
	root.addChildren(child1, child2, child3)
	err := miruken.TraverseAxis(root, miruken.TraverseDescendantReverse, suite)
	suite.Nil(err)
	suite.Visited(child31, child3, child2, child1)
}

func (suite *TraversingTestSuite) TestTraverseDescendantsAndSelf() {
	var root    = &treeNode{data: "root"}
	var child1  = &treeNode{data: "child1"}
	var child2  = &treeNode{data: "child2"}
	var child3  = &treeNode{data: "child3"}
	var child31 = &treeNode{data: "child31"}
	child3.addChildren(child31)
	root.addChildren(child1, child2, child3)
	err := miruken.TraverseAxis(root, miruken.TraverseSelfOrDescendant, suite)
	suite.Nil(err)
	suite.Visited(root, child1, child2, child3, child31)
}

func (suite *TraversingTestSuite) TestTraverseDescendantsAndSelfReverse() {
	var root    = &treeNode{data: "root"}
	var child1  = &treeNode{data: "child1"}
	var child2  = &treeNode{data: "child2"}
	var child3  = &treeNode{data: "child3"}
	var child31 = &treeNode{data: "child31"}
	child3.addChildren(child31)
	root.addChildren(child1, child2, child3)
	err := miruken.TraverseAxis(root, miruken.TraverseSelfOrDescendantReverse, suite)
	suite.Nil(err)
	suite.Visited(child31, child1, child2, child3, root)
}

func (suite *TraversingTestSuite) TestTraverseAncestorSiblingAndSelf() {
	var root    = &treeNode{data: "root"}
	var parent  = &treeNode{data: "parent"}
	var child1  = &treeNode{data: "child1"}
	var child2  = &treeNode{data: "child2"}
	var child3  = &treeNode{data: "child3"}
	var child31 = &treeNode{data: "child31"}
	child3.addChildren(child31)
	parent.addChildren(child1, child2, child3)
	root.addChildren(parent)
	err := miruken.TraverseAxis(child3, miruken.TraverseSelfSiblingOrAncestor, suite)
	suite.Nil(err)
	suite.Visited(child3, child1, child2, parent, root)
}

func TestTraversingTestSuite(t *testing.T) {
	suite.Run(t, new(TraversingTestSuite))
}
