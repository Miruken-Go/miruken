package test

import (
	"github.com/miruken-go/miruken/internal/slices"
	"github.com/stretchr/testify/suite"
	"testing"
)

type SafeTestSuite struct {
	suite.Suite
}

func (suite *SafeTestSuite) TestHelper() {
	suite.Run("Empty", func() {
		sl := slices.NewSafe[int]()
		suite.Len(sl.Items(), 0)
	})

	suite.Run("Initial", func() {
		sl := slices.NewSafe[string]("hello", "world")
		suite.ElementsMatch([]string{"hello", "world"}, sl.Items())
	})

	suite.Run("Index", func() {
		sl := slices.NewSafe[int](7, 12, 22, 100)
		suite.Equal(1, sl.Index(slices.Item(12)))
		suite.Equal(3, sl.Index(slices.Item(100)))
		suite.Equal(-1, sl.Index(slices.Item(23)))
	})

	suite.Run("Append", func() {
		sl := slices.NewSafe[int]()
		sl.Append(3, 6, 9)
		suite.ElementsMatch([]int{3, 6, 9}, sl.Items())
		sl.Append(5, 10)
		suite.ElementsMatch([]int{3, 6, 9, 5, 10}, sl.Items())
	})

	suite.Run("Insert", func() {
		sl := slices.NewSafe[string]()
		sl.Insert(0, "red", "green")
		suite.ElementsMatch([]string{"red", "green"}, sl.Items())
		sl.Insert(2, "blue", "purple")
		suite.ElementsMatch([]string{"red", "green", "blue", "purple"}, sl.Items())
		sl.Insert(1, "orange")
		suite.ElementsMatch([]string{"red", "orange", "green", "blue", "purple"}, sl.Items())
	})

	suite.Run("Delete", func() {
		sl := slices.NewSafe[int](2, 3, 14, 9, 36)
		sl.Delete(func(i int) (bool, bool) {
			return i == 3 || i == 14, false
		})
		suite.ElementsMatch([]int{2, 9, 36}, sl.Items())
		sl.Delete(func(i int) (bool, bool) {
			return i == 2 || i == 9, false
		})
		suite.ElementsMatch([]int{36}, sl.Items())
	})
}

func TestSafeTestSuite(t *testing.T) {
	suite.Run(t, new(SafeTestSuite))
}
