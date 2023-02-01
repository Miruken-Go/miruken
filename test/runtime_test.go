package test

import (
	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
	"reflect"
	"testing"
)

func Launch() {}
func dismiss() {}

type RuntimeTestSuite struct {
	suite.Suite
}

func (suite *RuntimeTestSuite) TestRuntime() {
	suite.Run("CopyIndirect", func () {
		suite.Run("Convert", func () {
			var f float32
			miruken.CopyIndirect(22, &f)
			suite.Equal(float32(22), f)

			var i int
			miruken.CopyIndirect(f, &i)
			suite.Equal(22, i)
		})
	})

	suite.Run("CopySliceIndirect", func () {
		suite.Run("Convert", func () {
			var f []float32
			miruken.CopySliceIndirect([]any{34}, &f)
		})
	})

	suite.Run("CoerceSlice", func () {
		suite.Run("Convert", func () {
			fa := []any{3.2, 19.9}
			sl, ok := miruken.CoerceSlice(reflect.ValueOf(fa), reflect.TypeOf(float32(1)))
			suite.True(ok)
			suite.Equal([]float32{3.2, 19.9}, sl.Interface())
		})
	})

	suite.Run("Exported", func () {
		suite.Run("Func", func () {
			suite.True(miruken.Exported(Launch))
			suite.False(miruken.Exported(dismiss))
		})
	})

}

func TestRuntimeTestSuite(t *testing.T) {
	suite.Run(t, new(RuntimeTestSuite))
}