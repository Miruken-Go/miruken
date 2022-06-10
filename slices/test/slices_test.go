package test

import (
	"github.com/miruken-go/miruken/slices"
	"github.com/stretchr/testify/suite"
	"strconv"
	"strings"
	"testing"
)

type SlicesTestSuite struct {
	suite.Suite
}

func (suite *SlicesTestSuite) TestSlices() {
	suite.Run("Map", func() {
		s := []string{"a", "b", "c"}
		result := slices.Map[string, string](s, func(i int, s string) string {
			return strings.ToUpper(s)
		})
		expected := []string{"A", "B", "C"}

		suite.Equal(expected, result)
		suite.Equal([]string{"FISH"}, slices.Map[string, string]([]string{"fish"}, strings.ToUpper))
		suite.Equal([]string{"fish"}, slices.Map[string, string]([]string{" fish "}, strings.TrimSpace))
		suite.Equal([]int{4}, slices.Map[string, int]([]string{"fish"}, func(s string) int { return len(s) }))
	})

	suite.Run("FlatMap", func() {
		s := []string{"a", "b", "c"}
		result := slices.FlatMap[string, string](s, func(i int, s string) []string {
			return []string{s, strings.ToUpper(s)}
		})
		expected := []string{"a", "A", "b", "B", "c", "C"}

		suite.Equal(expected, result)
		suite.Len(slices.FlatMap[string, string]([]string{"X", "Y", "Z"},
			func(s string) []string {
				return []string{}
			}), 0)
	})

	suite.Run("Filter", func() {
		result := slices.Filter([]string{"car1", "car2", "bus1", "bus2"}, func(i int, s string) bool {
			return strings.HasPrefix(s, "car")
		})
		suite.Equal([]string{"car1", "car2"}, result)
	})

	suite.Run("OfType", func() {
		var s = []any{1, "two", 3, "four"}
		result := slices.OfType[any, int](s)
		suite.Equal([]int{1, 3}, result)
		result2 := slices.OfType[any, string](s)
		suite.Equal([]string{"two", "four"}, result2)
	})

	suite.Run("Reduce", func() {
		s := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
		result := slices.Reduce(s, "0", func(acc string, i int, s string) string {
			accumulator, _ := strconv.Atoi(acc)
			current,     _ := strconv.Atoi(s)
			s = strconv.Itoa(accumulator + current)
			return s
		})
		suite.Equal("55", result)
	})

	suite.Run("First", func() {
		first, ok := slices.First([]int{1, 2, 3})
		suite.Equal(1, first)
		suite.Equal(true, ok)
		first2, ok := slices.First([]string{})
		suite.Equal( "", first2)
		suite.Equal( false, ok)
	})

	suite.Run("Last", func() {
		last, ok := slices.Last([]int{1, 2, 3})
		suite.Equal(3, last)
		suite.Equal( true, ok)
		last, ok = slices.Last([]int{})
		suite.Equal(0, last)
		suite.Equal(false, ok)
	})
}

func TestSlicesTestSuite(t *testing.T) {
	suite.Run(t, new(SlicesTestSuite))
}
