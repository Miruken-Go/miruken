package test

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/miruken-go/miruken/internal/seq"
	"github.com/stretchr/testify/suite"
)

type SeqTestSuite struct {
	suite.Suite
}

func (suite *SeqTestSuite) TestSequences() {
	suite.Run("Filter", func() {
		result := slices.Collect(
			seq.Filter(
				slices.Values([]string{"car1", "bus1", "bus2", "car2"}),
				func(s string) bool {
					return strings.HasPrefix(s, "car")
				}))
		suite.Equal([]string{"car1", "car2"}, result)
	})

	suite.Run("Filter2", func() {
		m := maps.Collect(
			seq.Filter2(
				slices.All([]string{"car1", "bus1", "bus2", "car2"}),
				func(i int, s string) bool {
					return strings.HasPrefix(s, "car")
				}))
		suite.True(maps.Equal(m, map[int]string{0: "car1", 1: "car2"}))
	})

	suite.Run("OfType", func() {
		result := slices.Collect(
			seq.OfType[any,int](slices.Values([]any{1, "two", 3, "four"})))
		suite.Equal([]int{1, 3}, result)

		result2 := slices.Collect(
			seq.OfType[any,string](slices.Values([]any{1, "two", 3, "four"})))
		suite.Equal([]string{"two", "four"}, result2)
	})

	suite.Run("OfType2", func() {
		m := maps.Collect(seq.OfType2[any,int](slices.Values([]any{1, "two", 3, "four"})))
		suite.True(maps.Equal(m, map[int]int{0: 1, 1: 3}))
	})

	suite.Run("Map", func() {
		result := slices.Collect(seq.Map(slices.Values([]string{"a", "b", "c"}), strings.ToUpper))
		suite.Equal([]string{"A", "B", "C"}, result)

		suite.Equal([]string{"fish"}, slices.Collect(seq.Map(slices.Values([]string{" fish "}), strings.TrimSpace)))
		suite.Equal([]int{4}, slices.Collect(seq.Map(slices.Values([]string{"fish"}), func(s string) int { return len(s) })))
	})

	suite.Run("Map2", func() {
		m := maps.Collect(
			seq.Map2(slices.All([]string{"a", "b", "c"}),
				func(i int, s string) string {
					return strings.ToUpper(s)
				}))
		suite.True(maps.Equal(m, map[int]string{0: "A", 1: "B", 2: "C"}))
	})

	suite.Run("FlatMap", func() {
		result := slices.Collect(seq.FlatMap(
			slices.Values([]string{"a", "b", "c"}),
			func(s string) []string {
				return []string{s, strings.ToUpper(s)}
			}))
		suite.Equal([]string{"a", "A", "b", "B", "c", "C"}, result)

		result = slices.Collect(seq.FlatMap(
			slices.Values([]string{"X", "Y", "Z"}),
			func(s string) []string {
				return []string{}
			}))
		suite.Len(result, 0)
	})

	suite.Run("FlatMap2", func() {
		m := maps.Collect(seq.FlatMap2(
			slices.All([]string{"a", "b", "c"}),
			func(i int, s string) []string {
				return []string{s, strings.ToUpper(s)}
			}))
		suite.True(maps.Equal(m, map[int]string{0: "a", 1: "A", 2: "b", 3: "B", 4: "c", 5: "C"}))
	})

	suite.Run("Remove", func() {
		result := slices.Collect(seq.Except(
			slices.Values([]string{"car1", "bus1", "bus2", "car2"}),
			slices.Values([]string{"bus1", "car1"})),
		)
		suite.ElementsMatch([]string{"car2", "bus2"}, result)
	})

	suite.Run("First", func() {
		first, ok := seq.First(slices.Values([]int{1, 2, 3}))
		suite.Equal(1, first)
		suite.Equal(true, ok)
		first2, ok :=  seq.First(slices.Values([]string{}))
		suite.Equal("", first2)
		suite.Equal(false, ok)
	})

	suite.Run("Last", func() {
		last, ok := seq.Last(slices.Values([]int{1, 2, 3}))
		suite.Equal(3, last)
		suite.Equal(true, ok)
		last2, ok :=  seq.First(slices.Values([]string{}))
		suite.Equal("", last2)
		suite.Equal(false, ok)
	})

	suite.Run("Reduce", func() {
		s := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
		result := seq.Reduce(
			slices.Values(s),
			func(acc string, s string) string {
			accumulator, _ := strconv.Atoi(acc)
			current, _ := strconv.Atoi(s)
			s = strconv.Itoa(accumulator + current)
			return s
		}, "0")
		suite.Equal("55", result)
	})
}

func TestSeqTestSuite(t *testing.T) {
	suite.Run(t, new(SeqTestSuite))
}
