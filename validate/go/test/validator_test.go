package test

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/validate"
	"github.com/miruken-go/miruken/validate/go"
	"github.com/stretchr/testify/suite"
	"testing"
)

//go:generate $GOPATH/bin/miruken -tests

// Address contains user address information.
type Address struct {
	Street string `valid:"required"`
	Zip    string `valid:"numeric,required"`
}

// User contains user information.
type User struct {
	Id       int
	Name     string `valid:"required"`
	Email    string `valid:"required,email"`
	Password string `valid:"required"`
	Age      int    `valid:"required"`
	Home     *Address
	Work     []Address `valid:"required"`
}

// Command to create a User.
type CreateUser struct {
	User User
}

// UserHandler handles User commands.
type UserHandler struct {
	userId int
}

func (u *UserHandler) CreateUser(
	_ *miruken.Handles, create *CreateUser,
) User {
	user := create.User
	u.userId++
	user.Id = u.userId
	return user
}

type ValidatorTestSuite struct {
	suite.Suite
	handler miruken.Handler
}

func (suite *ValidatorTestSuite) SetupTest() {
	suite.handler, _ = miruken.Setup(
		TestFeature,
		govalidator.Feature(),
	)
}

func (suite *ValidatorTestSuite) TestValidator() {
	suite.Run("Valid Target", func() {
		create := CreateUser{
			User{
				Name:     "John",
				Email:    "john@yahoo.com",
				Password: "123G#678",
				Age:      20,
				Home: &Address{
					Street: "Street",
					Zip:    "123456",
				},
				Work: []Address{{
					Street: "Street",
					Zip:    "123456",
				}, {
					Street: "Street",
					Zip:    "123456",
				}},
			},
		}
		if user, _, err := miruken.Execute[User](suite.handler, &create); err == nil {
			suite.Equal(1, user.Id)
		} else {
			suite.Fail("unexpected error", err.Error())
		}
	})

	suite.Run("Invalid Target", func() {
		create := CreateUser{
			User{
				Email: "john",
				Home:  &Address{},
				Work:  []Address{{}},
			},
		}
		if _, _, err := miruken.Execute[User](suite.handler, &create); err != nil {
			suite.IsType(&validate.Outcome{}, err)
			outcome := err.(*validate.Outcome)
			suite.False(outcome.Valid())
			user := outcome.Path("User")
			suite.Equal("Age: User.Age: non zero value required; Email: User.Email: john does not validate as email; Home: (Street: User.Home.Street: non zero value required; Zip: User.Home.Zip: non zero value required); Name: User.Name: non zero value required; Password: User.Password: non zero value required; Work: (0: (Street: User.Work.0.Street: non zero value required; Zip: User.Work.0.Zip: non zero value required))", user.Error())
		} else {
			suite.Fail("expected error")
		}
	})

	suite.Run("Group", func() {
		create := CreateUser{
			User{
				Email: "john",
				Home:  &Address{},
				Work:  []Address{{}},
			},
		}
		outcome, _, err := validate.Validate(suite.handler, &create, "Admin")
		suite.Nil(err)
		suite.NotNil(outcome)
		suite.False(outcome.Valid())
		suite.Equal([]string{"User"}, outcome.Fields())
		suite.Equal("User: (Age: User.Age: non zero value required; Email: User.Email: john does not validate as email; Home: (Street: User.Home.Street: non zero value required; Zip: User.Home.Zip: non zero value required); Name: User.Name: non zero value required; Password: User.Password: non zero value required; Work: (0: (Street: User.Work.0.Street: non zero value required; Zip: User.Work.0.Zip: non zero value required)))", outcome.Error())
	})
}

func TestValidateTestSuite(t *testing.T) {
	suite.Run(t, new(ValidatorTestSuite))
}
