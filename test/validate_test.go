package test

import (
	"errors"
	"github.com/bearbin/go-age"
	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
	"reflect"
	"testing"
	"time"
)

type Model struct {
	outcome *miruken.ValidationOutcome
}

func (m *Model) ValidationOutcome() *miruken.ValidationOutcome {
	return m.outcome
}

func (m *Model) SetValidationOutcome(outcome *miruken.ValidationOutcome) {
	m.outcome = outcome
}

type Player struct {
	Model
	FirstName string
	LastName  string
	DOB       time.Time
}

type Coach struct {
	Model
	FirstName string
	LastName  string
	License   string
}

type Team struct {
	Id         int
	Active     bool
	Name       string
	Division   string
	Coach      *Coach
	Players    []*Player
	Registered bool
}

// PlayerValidator
type PlayerValidator struct{}

func (p *PlayerValidator) ShouldHaveFullName(
	validates *miruken.Validates, player *Player,
) {
	outcome := validates.Outcome()

	if len(player.FirstName) == 0 {
		outcome.AddError("FirstName", errors.New(`"First Name" is required`))
	}

	if len(player.FirstName) == 0 {
		outcome.AddError("LastName", errors.New(`"Last Name" is required`))
	}

	if player.DOB.IsZero() {
		outcome.AddError("DOB", errors.New(`"DOB" is required`))
	}
}

func (p *PlayerValidator) MustBeTenOrUnder(
	_ *struct{
		miruken.Validates
		miruken.Scope `scope:"Recreational"`
	  }, player *Player,
	validates *miruken.Validates,
) {
	if dob := player.DOB; !dob.IsZero() {
		if age.Age(dob) > 10 {
			validates.Outcome().AddError("DOB",
				errors.New("player must be 10 years old or younger"))
		}
	}
}

type ValidateTestSuite struct {
	suite.Suite
	HandleTypes []reflect.Type
}

func (suite *ValidateTestSuite) SetupTest() {
	handleTypes := []reflect.Type{
		miruken.TypeOf[*PlayerValidator](),
	}
	suite.HandleTypes = handleTypes
}

func (suite *ValidateTestSuite) InferenceRoot() miruken.Handler {
	return miruken.NewRootHandler(miruken.WithHandlerTypes(suite.HandleTypes...))
}

func (suite *ValidateTestSuite) InferenceRootWith(
	handlerTypes ... reflect.Type) miruken.Handler {
	return miruken.NewRootHandler(miruken.WithHandlerTypes(handlerTypes...))
}

func (suite *ValidateTestSuite) TestValidation() {
	suite.Run("ValidationOutcome", func () {
		suite.Run("Simple Error", func() {
			outcome := &miruken.ValidationOutcome{}
			outcome.AddError("Name", errors.New(`"Name" can't be empty`))
			suite.Equal(`Name: "Name" can't be empty`, outcome.Error())
			suite.Equal([]string{"Name"}, outcome.Culprits())
		})

		suite.Run("Nested Error", func() {
			outcome := &miruken.ValidationOutcome{}
			outcome.AddError("Company.Name", errors.New(`"Name" can't be empty`))
			suite.Equal(`Company: (Name: "Name" can't be empty)`, outcome.Error())
			suite.Equal([]string{"Company"}, outcome.Culprits())
			errors := outcome.PathErrors("Company")
			suite.Len(errors, 1)
			suite.IsType(&miruken.ValidationOutcome{}, errors[0])
			company := errors[0].(*miruken.ValidationOutcome)
			suite.Equal(`Name: "Name" can't be empty`, company.Error())
			suite.Equal([]string{"Name"}, company.Culprits())
		})

		suite.Run("Mixed Errors", func() {
			outcome := &miruken.ValidationOutcome{}
			outcome.AddError("Name", errors.New(`"Name" can't be empty`))
			outcome.AddError("Company.Name", errors.New(`"Name" can't be empty`))
			suite.Equal(`Company: (Name: "Name" can't be empty); Name: "Name" can't be empty`, outcome.Error())
			suite.ElementsMatch([]string{"Name", "Company"}, outcome.Culprits())
		})
	})

	suite.Run("Validates", func () {
		suite.Run("Default", func() {
			handler := suite.InferenceRoot()
			player  := Player{DOB:  time.Date(2007, time.June,
				14, 13, 26, 00, 0, time.Local) }
			outcome, err := miruken.Validate(handler, &player)
			suite.Nil(err)
			suite.NotNil(outcome)
			suite.False(outcome.Valid())
			suite.Same(outcome, player.ValidationOutcome())
			suite.ElementsMatch([]string{"FirstName", "LastName"}, outcome.Culprits())
			suite.Equal(`FirstName: "First Name" is required; LastName: "Last Name" is required`, outcome.Error())
		})

		suite.Run("Scope", func() {
			handler := suite.InferenceRoot()
			player  := Player{
				FirstName: "Matthew",
				LastName:  "Dudley",
				DOB:       time.Date(2007, time.June, 14,
					13, 26, 00, 0, time.Local),
			}
			outcome, err := miruken.Validate(handler, &player, "Recreational")
			suite.Nil(err)
			suite.NotNil(outcome)
			suite.False(outcome.Valid())
			suite.Same(outcome, player.ValidationOutcome())
			suite.Equal([]string{"DOB"}, outcome.Culprits())
			suite.Equal("DOB: player must be 10 years old or younger", outcome.Error())
		})
	})
}

func TestValidateTestSuite(t *testing.T) {
	suite.Run(t, new(ValidateTestSuite))
}
