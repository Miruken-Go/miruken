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
	Coach      Coach
	Players    []Player
	Registered bool
}

type TeamAction struct {
	Model
	Team Team
}

type CreateTeam struct {
	TeamAction
}

type RemoveTeam struct {
	TeamAction
}

func (c *CreateTeam) ValidateMe(
	validates *miruken.Validates,
) {
	if c.Team.Name == "Breakaway" {
		validates.Outcome().
			AddError("Name", errors.New(`"Breakaway" is a reserved name`))
	}
}

// PlayerValidator
type PlayerValidator struct{}

func (v *PlayerValidator) MustHaveNameAndDOB(
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

func (v *PlayerValidator) MustBeTenOrUnder(
	_*struct{
		miruken.Validates
		miruken.Group `name:"Recreational"`
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

// TeamValidator
type TeamValidator struct{}

func (v *TeamValidator) MustHaveName(
	validates *miruken.Validates, team *Team,
) {
	if name := team.Name; len(name) == 0 {
		validates.Outcome().AddError("Name", errors.New(`"Name" is required`))
	}
}

func (v *TeamValidator) MustHaveLicensedCoach(
	_*struct{
		miruken.Validates
		miruken.Group `name:"ECNL"`
	  }, team *Team,
	validates *miruken.Validates,
) {
	outcome := validates.Outcome()

	if coach := team.Coach; reflect.ValueOf(coach).IsZero() {
		outcome.AddError("Coach", errors.New(`"Coach" is required`))
	} else if license := coach.License; len(license) == 0 {
		outcome.AddError("Coach.License", errors.New("licensed Coach is required"))
	}
}

func (v *TeamValidator) CreateTeam(
	validates *miruken.Validates, create *CreateTeam,
) {
	team := &create.Team
	v.MustHaveName(validates, team)
	if validates.InGroup("ECNL") {
		v.MustHaveLicensedCoach(nil, team, validates)
	}
}

func (v *TeamValidator) RemoveTeam(
	validates *miruken.Validates, remove *RemoveTeam,
) {
	if remove.Team.Id <= 0 {
		outcome := validates.Outcome()
		outcome.AddError("Id", errors.New(`"Id" must be greater than 0`))
	}
}

// OpenValidator
type OpenValidator struct {}

func (v *OpenValidator) Validate(
	validates *miruken.Validates, target any,
) {
	if v, ok := target.(interface {
		ValidateMe(*miruken.Validates)
	}); ok {
		v.ValidateMe(validates)
	}
}

type TeamHandler struct {
	teamId int
}

func (h *TeamHandler) CreateTeam(
	_*miruken.Handles, create *CreateTeam,
) Team {
	team := create.Team
	h.teamId++
	team.Id     = h.teamId
	team.Active = true
	return team
}

func (h *TeamHandler) RemoveTeam(
	_*miruken.Handles, remove *RemoveTeam,
) Team {
	team := remove.Team
	team.Active = false
	return team
}

type ValidatesTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *ValidatesTestSuite) SetupTest() {
	suite.specs = []any{
		&OpenValidator{},
		&PlayerValidator{},
		&TeamValidator{},
		&TeamHandler{},
	}
}

func (suite *ValidatesTestSuite) Setup() (miruken.Handler, error) {
	return suite.SetupWith(suite.specs...)
}

func (suite *ValidatesTestSuite) SetupWith(specs ... any) (miruken.Handler, error) {
	return miruken.Setup(miruken.HandlerSpecs(specs...))
}

func (suite *ValidatesTestSuite) TestValidation() {
	suite.Run("ValidationOutcome", func () {
		suite.Run("Root Errors", func() {
			outcome := &miruken.ValidationOutcome{}
			outcome.AddError("", errors.New("player not found"))
			suite.Equal(": player not found", outcome.Error())
			suite.Equal([]string{""}, outcome.Fields())
			suite.ElementsMatch(
				[]error{errors.New("player not found")},
				outcome.FieldErrors(""))
		})

		suite.Run("Simple Errors", func() {
			outcome := &miruken.ValidationOutcome{}
			outcome.AddError("Name", errors.New(`"Name" can't be empty`))
			suite.Equal(`Name: "Name" can't be empty`, outcome.Error())
			suite.Equal([]string{"Name"}, outcome.Fields())
			suite.ElementsMatch(
				[]error{errors.New(`"Name" can't be empty`)},
				outcome.FieldErrors("Name"))
		})

		suite.Run("Nested Errors", func() {
			outcome := &miruken.ValidationOutcome{}
			outcome.AddError("Company.Name", errors.New(`"Name" can't be empty`))
			suite.Equal(`Company: (Name: "Name" can't be empty)`, outcome.Error())
			suite.Equal([]string{"Company"}, outcome.Fields())
			company := outcome.Path("Company")
			suite.Equal(`Name: "Name" can't be empty`, company.Error())
			suite.Equal([]string{"Name"}, company.Fields())
			suite.ElementsMatch(
				[]error{errors.New(`"Name" can't be empty`)},
				outcome.FieldErrors("Company.Name"))
		})

		suite.Run("Mixed Errors", func() {
			outcome := &miruken.ValidationOutcome{}
			outcome.AddError("Name", errors.New(`"Name" can't be empty`))
			outcome.AddError("Company.Name", errors.New(`"Name" can't be empty`))
			suite.Equal(`Company: (Name: "Name" can't be empty); Name: "Name" can't be empty`, outcome.Error())
			suite.ElementsMatch([]string{"Name", "Company"}, outcome.Fields())
			suite.ElementsMatch(
				[]error{errors.New(`"Name" can't be empty`)},
				outcome.FieldErrors("Name"))
			suite.ElementsMatch(
				[]error{errors.New(`"Name" can't be empty`)},
				outcome.FieldErrors("Company.Name"))
		})

		suite.Run("Collection Errors", func() {
			outcome := &miruken.ValidationOutcome{}
			outcome.AddError("Players[0]", errors.New(`"Players[0]" can't be empty`))
			suite.Equal(`Players: (0: "Players[0]" can't be empty)`, outcome.Error())
			suite.Equal([]string{"Players"}, outcome.Fields())
			players := outcome.Path("Players")
			suite.Equal(`0: "Players[0]" can't be empty`, players.Error())
			suite.ElementsMatch(
				[]error{errors.New(`"Players[0]" can't be empty`)},
				outcome.FieldErrors("Players[0]"))
			suite.ElementsMatch(
				[]error{errors.New(`"Players[0]" can't be empty`)},
				outcome.FieldErrors("Players.0"))
		})

		suite.Run("Cannot add path outcome directly", func() {
			defer func() {
				if r := recover(); r != nil {
					suite.Equal("cannot add path ValidationOutcome directly", r)
				}
			}()
			outcome := &miruken.ValidationOutcome{}
			outcome.AddError("Foo", &miruken.ValidationOutcome{})
			suite.Fail("Expected panic")
		})
	})

	suite.Run("Validates", func () {
		suite.Run("Default", func() {
			handler, _ := suite.Setup()
			player := Player{DOB:  time.Date(2007, time.June,
				14, 13, 26, 00, 0, time.Local) }
			outcome, _, err := miruken.Validate(handler, &player)
			suite.Nil(err)
			suite.NotNil(outcome)
			suite.False(outcome.Valid())
			suite.Same(outcome, player.ValidationOutcome())
			suite.ElementsMatch([]string{"FirstName", "LastName"}, outcome.Fields())
			suite.Equal(`FirstName: "First Name" is required; LastName: "Last Name" is required`, outcome.Error())
		})

		suite.Run("Group", func() {
			handler, _ := suite.Setup()
			player := Player{
				FirstName: "Matthew",
				LastName:  "Dudley",
				DOB:       time.Date(2007, time.June, 14,
					13, 26, 00, 0, time.Local),
			}
			outcome, _, err := miruken.Validate(handler, &player, miruken.Rules("Recreational"))
			suite.Nil(err)
			suite.NotNil(outcome)
			suite.False(outcome.Valid())
			suite.Same(outcome, player.ValidationOutcome())
			suite.Equal([]string{"DOB"}, outcome.Fields())
			suite.Equal("DOB: player must be 10 years old or younger", outcome.Error())
		})
	})

	suite.Run("ValidateFilter", func () {
		handler, _ := suite.Setup()
		var handles miruken.Handles
		validatesFilter := miruken.NewValidateProvider(true)
		handles.Policy().AddFilters(validatesFilter)
		defer func() {
			handles.Policy().RemoveFilters(validatesFilter)
		}()

		suite.Run("Validates Command", func() {
			create := CreateTeam{TeamAction{ Team: Team{
				Name: "Liverpool",
				Coach: Coach{
					FirstName: "Zinedine",
					LastName:  "Zidane",
					License:   "A",
				},
			}}}
			if team, _, err := miruken.Execute[Team](handler, &create); err == nil {
				suite.Equal(1, team.Id)
				suite.True(team.Active)
				outcome := create.ValidationOutcome()
				suite.NotNil(outcome)
				suite.True(outcome.Valid())
			} else {
				suite.Fail("unexpected error: %v", err.Error())
			}
		})

		suite.Run("Rejects Command", func() {
			var create CreateTeam
			if team, _, err := miruken.Execute[Team](handler, &create); err != nil {
				suite.IsType(&miruken.ValidationOutcome{}, err)
				suite.Equal(0, team.Id)
				outcome := create.ValidationOutcome()
				suite.NotNil(outcome)
				suite.False(outcome.Valid())
				suite.Equal(`Name: "Name" is required`, outcome.Error())
			} else {
				suite.Fail("expected validation error")
			}
		})

		suite.Run("Rejects Another Command", func() {
			remove := &RemoveTeam{}
			if team, _, err := miruken.Execute[Team](handler, remove); err != nil {
				suite.IsType(&miruken.ValidationOutcome{}, err)
				suite.False(team.Active)
				outcome := remove.ValidationOutcome()
				suite.NotNil(outcome)
				suite.False(outcome.Valid())
				suite.Equal(`Id: "Id" must be greater than 0`, outcome.Error())
			} else {
				suite.Fail("unexpected error: %v", err.Error())
			}

			suite.Run("Validates Open", func() {
				create := CreateTeam{TeamAction{ Team: Team{
					Name: "Breakaway",
					Coach: Coach{
						FirstName: "Frank",
						LastName:  "Lampaerd",
						License:   "B",
					},
				}}}
				if _, _, err := miruken.Execute[Team](handler, &create); err != nil {
					suite.IsType(&miruken.ValidationOutcome{}, err)
					outcome := create.ValidationOutcome()
					suite.NotNil(outcome)
					suite.False(outcome.Valid())
					suite.Equal(`Name: "Breakaway" is a reserved name`, outcome.Error())
				} else {
					suite.Fail("expected validation error")
				}
			})
		})
	})
}

func TestValidatesTestSuite(t *testing.T) {
	suite.Run(t, new(ValidatesTestSuite))
}
