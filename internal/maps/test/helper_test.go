package test

import (
	"testing"

	"github.com/miruken-go/miruken/internal/maps"
	"github.com/stretchr/testify/suite"
)

type HelperTestSuite struct {
	suite.Suite
}

func (suite *HelperTestSuite) TestHelper() {
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

func TestHelperTestSuite(t *testing.T) {
	suite.Run(t, new(HelperTestSuite))
}
