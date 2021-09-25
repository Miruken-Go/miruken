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

func (t *treeNode) addChildren(children ... *treeNode) {
	for _, child := range  children {
		t.children = append(t.children, child)
	}
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
