package test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/miruken-go/miruken/internal"
	"github.com/stretchr/testify/suite"
)

func Launch()  {}
func dismiss() {}

type RuntimeTestSuite struct {
	suite.Suite
}

func (suite *RuntimeTestSuite) TestRuntime() {
	suite.Run("CopyIndirect", func() {
		suite.Run("Convert", func() {
			var f float32
			internal.CopyIndirect(22, &f)
			suite.Equal(float32(22), f)

			var i int
			internal.CopyIndirect(f, &i)
			suite.Equal(22, i)
		})
	})

	suite.Run("CopySliceIndirect", func() {
		suite.Run("Convert", func() {
			var f []float32
			internal.CopySliceIndirect([]any{34}, &f)
		})
	})

	suite.Run("CoerceSlice", func() {
		suite.Run("Convert", func() {
			fa := []any{3.2, 19.9}
			sl, ok := internal.CoerceSlice(reflect.ValueOf(fa), reflect.TypeOf(float32(1)))
			suite.True(ok)
			suite.Equal([]float32{3.2, 19.9}, sl.Interface())
		})
	})

	suite.Run("Exported", func() {
		suite.Run("Func", func() {
			suite.True(internal.Exported(Launch))
			suite.False(internal.Exported(dismiss))
		})
	})

	suite.Run("CombineStructTags", func() {
		tests := []struct {
			name     string
			tags     []reflect.StructTag
			expected reflect.StructTag
		}{
			{
				name:     "Zero tags",
				tags:     []reflect.StructTag{},
				expected: reflect.StructTag(""),
			},
			{
				name:     "One tag",
				tags:     []reflect.StructTag{`json:"name"`},
				expected: reflect.StructTag(`json:"name"`),
			},
			{
				name: "Multiple tags with no duplicates",
				tags: []reflect.StructTag{
					`json:"name"`,
					`xml:"name"`,
				},
				expected: reflect.StructTag(`json:"name" xml:"name"`),
			},
			{
				name: "Multiple tags with duplicates, last one wins",
				tags: []reflect.StructTag{
					`json:"name" xml:"name"`,
					`yaml:"name" xml:"title"`,
				},
				expected: reflect.StructTag(`json:"name" yaml:"name" xml:"title"`),
			},
			{
				name: "Empty tags included",
				tags: []reflect.StructTag{
					`json:"name"`,
					``,
					`xml:"name"`,
				},
				expected: reflect.StructTag(`json:"name" xml:"name"`),
			},
		}

		for _, tt := range tests {
			suite.T().Run(tt.name, func(t *testing.T) {
				actual := internal.CombineStructTags(tt.tags...)
				if !compareStructTags(actual, tt.expected) {
					t.Errorf("CombineStructTags() = %v, expected %v", actual, tt.expected)
				}
			})
		}
	})
}

func TestRuntimeTestSuite(t *testing.T) {
	suite.Run(t, new(RuntimeTestSuite))
}

func parseStructTag(tag reflect.StructTag) map[string]string {
	result := make(map[string]string)
	if tag == "" {
		return result
	}
	tagString := string(tag)
	tagParts := strings.Split(tagString, " ")
	for _, part := range tagParts {
		keyValue := strings.SplitN(part, ":", 2)
		if len(keyValue) == 2 {
			key := keyValue[0]
			value := keyValue[1]
			result[key] = value
		}
	}
	return result
}

func compareStructTags(tag1, tag2 reflect.StructTag) bool {
	map1 := parseStructTag(tag1)
	map2 := parseStructTag(tag2)
	if len(map1) != len(map2) {
		return false
	}
	for key, value := range map1 {
		if map2[key] != value {
			return false
		}
	}
	return true
}