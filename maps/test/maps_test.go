package test

import (
	"github.com/miruken-go/miruken/maps"
	"github.com/stretchr/testify/suite"
	"testing"
)

type MapsTestSuite struct {
	suite.Suite
}

func (suite *MapsTestSuite) TestMaps() {
	suite.Run("Keys", func() {
		months := map[int]string{
			7:  "Craig",
			28: "Brenda",
			9:  "Kaitlyn",
			29: "Lauren",
			14: "Matthew",
		}
		keys := maps.Keys(months)
		suite.ElementsMatch([]int{7, 9, 14, 28, 29}, keys)
	})
}

func TestMapsTestSuite(t *testing.T) {
	suite.Run(t, new(MapsTestSuite))
}
