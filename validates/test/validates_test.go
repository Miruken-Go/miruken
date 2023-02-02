package test

import (
	"errors"
	"github.com/bearbin/go-age"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/validates"
	"github.com/stretchr/testify/suite"
	"reflect"
	"testing"
	"time"
)

type Model struct {
	outcome *validates.Outcome
}

func (m *Model) ValidationOutcome() *validates.Outcome {
	return m.outcome
}

func (m *Model) SetValidationOutcome(outcome *validates.Outcome) {
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

func (c *CreateTeam) ValidateMe(it *validates.It) {
	if c.Team.Name == "Breakaway" {
		it.Outcome().
			AddError("Name", errors.New(`"Breakaway" is a reserved name`))
	}
}

// PlayerValidator
type PlayerValidator struct{}

func (v *PlayerValidator) MustHaveNameAndDOB(
	it *validates.It, player *Player,
) {
	outcome := it.Outcome()

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
		validates.It
		validates.Group `name:"Recreational"`
	  }, player *Player,
	it *validates.It,
) {
	if dob := player.DOB; !dob.IsZero() {
		if age.Age(dob) > 10 {
			it.Outcome().AddError("DOB",
				errors.New("player must be 10 years old or younger"))
		}
	}
}

// TeamValidator
type TeamValidator struct{}

func (v *TeamValidator) MustHaveName(
	it *validates.It, team *Team,
) {
	if name := team.Name; len(name) == 0 {
		it.Outcome().AddError("Name", errors.New(`"Name" is required`))
	}
}

func (v *TeamValidator) MustHaveLicensedCoach(
	_*struct{
		validates.It
		validates.Group `name:"ECNL"`
	  }, team *Team,
	it *validates.It,
) {
	outcome := it.Outcome()

	if coach := team.Coach; reflect.ValueOf(coach).IsZero() {
		outcome.AddError("Coach", errors.New(`"Coach" is required`))
	} else if license := coach.License; len(license) == 0 {
		outcome.AddError("Coach.License", errors.New("licensed Coach is required"))
	}
}

func (v *TeamValidator) CreateTeam(
	it *validates.It, create *CreateTeam,
) {
	team := &create.Team
	v.MustHaveName(it, team)
	if it.InGroup("ECNL") {
		v.MustHaveLicensedCoach(nil, team, it)
	}
}

func (v *TeamValidator) RemoveTeam(
	it *validates.It, remove *RemoveTeam,
) {
	if remove.Team.Id <= 0 {
		outcome := it.Outcome()
		outcome.AddError("Id", errors.New(`"Id" must be greater than 0`))
	}
}

// OpenValidator
type OpenValidator struct {}

func (v *OpenValidator) Validate(
	it *validates.It, target any,
) {
	if v, ok := target.(interface {
		ValidateMe(*validates.It)
	}); ok {
		v.ValidateMe(it)
	}
}

type TeamHandler struct {
	teamId int
}

func (h *TeamHandler) CreateTeam(
	_ *handles.It, create *CreateTeam,
) Team {
	team := create.Team
	h.teamId++
	team.Id     = h.teamId
	team.Active = true
	return team
}

func (h *TeamHandler) RemoveTeam(
	_ *handles.It, remove *RemoveTeam,
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

func (suite *ValidatesTestSuite) SetupWith(specs ...any) (miruken.Handler, error) {
	return miruken.Setup().Specs(specs...).Handler()
}

func (suite *ValidatesTestSuite) TestValidation() {
	suite.Run("Outcome", func () {
		suite.Run("Root Errors", func() {
			outcome := &validates.Outcome{}
			outcome.AddError("", errors.New("player not found"))
			suite.Equal(": player not found", outcome.Error())
			suite.Equal([]string{""}, outcome.Fields())
			suite.ElementsMatch(
				[]error{errors.New("player not found")},
				outcome.FieldErrors(""))
		})

		suite.Run("Simple Errors", func() {
			outcome := &validates.Outcome{}
			outcome.AddError("Name", errors.New(`"Name" can't be empty`))
			suite.Equal(`Name: "Name" can't be empty`, outcome.Error())
			suite.Equal([]string{"Name"}, outcome.Fields())
			suite.ElementsMatch(
				[]error{errors.New(`"Name" can't be empty`)},
				outcome.FieldErrors("Name"))
		})

		suite.Run("Nested Errors", func() {
			outcome := &validates.Outcome{}
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
			outcome := &validates.Outcome{}
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
			outcome := &validates.Outcome{}
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
					suite.Equal("cannot add path Outcome directly", r)
				}
			}()
			outcome := &validates.Outcome{}
			outcome.AddError("Foo", &validates.Outcome{})
			suite.Fail("Expected panic")
		})
	})

	suite.Run("It", func () {
		suite.Run("Default", func() {
			handler, _ := suite.Setup()
			player := Player{DOB:  time.Date(2007, time.June,
				14, 13, 26, 00, 0, time.Local) }
			outcome, _, err := validates.Validate(handler, &player)
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
			outcome, _, err := validates.Validate(handler, &player, validates.Groups("Recreational"))
			suite.Nil(err)
			suite.NotNil(outcome)
			suite.False(outcome.Valid())
			suite.Same(outcome, player.ValidationOutcome())
			suite.Equal([]string{"DOB"}, outcome.Fields())
			suite.Equal("DOB: player must be 10 years old or younger", outcome.Error())
		})
	})

	suite.Run("ValidateFilter", func () {
		handler, _ := miruken.Setup(
			validates.Feature(validates.Output)).
			Specs(suite.specs...).
			Handler()
		suite.Run("It Command", func() {
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
				suite.IsType(&validates.Outcome{}, err)
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
				suite.IsType(&validates.Outcome{}, err)
				suite.False(team.Active)
				outcome := remove.ValidationOutcome()
				suite.NotNil(outcome)
				suite.False(outcome.Valid())
				suite.Equal(`Id: "Id" must be greater than 0`, outcome.Error())
			} else {
				suite.Fail("unexpected error: %v", err.Error())
			}

			suite.Run("It Open", func() {
				create := CreateTeam{TeamAction{ Team: Team{
					Name: "Breakaway",
					Coach: Coach{
						FirstName: "Frank",
						LastName:  "Lampaerd",
						License:   "B",
					},
				}}}
				if _, _, err := miruken.Execute[Team](handler, &create); err != nil {
					suite.IsType(&validates.Outcome{}, err)
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
