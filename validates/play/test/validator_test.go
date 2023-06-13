package test

import (
	ut "github.com/go-playground/universal-translator"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/validates"
	play "github.com/miruken-go/miruken/validates/play"
	"github.com/stretchr/testify/suite"
	"testing"
)

//go:generate $GOPATH/bin/miruken -tests

// Address contains user address information.
type Address struct {
	Street string `validate:"required"`
	City   string `validate:"required"`
	Planet string `validate:"required"`
	Phone  string `validate:"required"`
}

// AddressNoTags contains user address information without tags.
type AddressNoTags struct {
	Street string
	City   string
	Planet string
	Phone  string
}

// User contains user information.
type User struct {
	Id             int       `validate:"eq=0"`
	FirstName      string    `validate:"required"`
	LastName       string    `validate:"required"`
	Age            uint8     `validate:"gte=0,lte=130"`
	Email          string    `validate:"required,email"`
	FavouriteColor string    `validate:"iscolor"`
	Addresses      []Address `validate:"required,dive"`
}

// UserNoTags contains user information without tags.
type UserNoTags struct {
	Id             int
	FirstName      string
	LastName       string
	Age            uint8
	Email          string
	FavouriteColor string
	Addresses      []AddressNoTags
}

// Command to create a User.
type CreateUser struct {
	User User
}

// Command to create a User without tags.
type CreateUserNoTags struct {
	User UserNoTags
}

// CreateUserIntegrity validates CreateUser
type CreateUserIntegrity struct {
	play.ValidatorT[*CreateUserNoTags]
}

// UserHandler handles User commands.
type UserHandler struct {
	userId int
}


// CreateUserIntegrity

func (v *CreateUserIntegrity) Constructor(
	_ *struct{args.Optional}, translator ut.Translator,
) error {
	return v.ConstructWithRules(
		play.Rules{
			{AddressNoTags{}, map[string]string{
				"Street": "required",
				"City":   "required",
				"Planet": "required",
				"Phone":  "required",
			}},
			{UserNoTags{}, map[string]string{
				"Id":             "eq=0",
				"FirstName":      "required",
				"LastName":       "required",
				"Age":            "gte=0,lte=130",
				"Email":          "required,email",
				"FavouriteColor": "iscolor",
				"Addresses":      "required,dive",
			}},
		}, nil, translator)
}


// UserHandler

func (u *UserHandler) CreateUser(
	_ *handles.It, create *CreateUser,
) User {
	user := create.User
	u.userId++
	user.Id = u.userId
	return user
}

func (u *UserHandler) CreateUserNoTags(
	_ *handles.It, create *CreateUserNoTags,
) UserNoTags {
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
		play.Feature(),
	).Handler()
}

func (suite *ValidatorTestSuite) TestValidator() {
	suite.Run("Tags", func() {
		suite.Run("Valid ForSource", func() {
			create := CreateUser{
				User{
					FirstName:      "Badger",
					LastName:       "Smith",
					Age:            52,
					Email:          "Badger.Smith@gmail.com",
					FavouriteColor: "#000",
					Addresses: []Address{
						{
							Street: "Eavesdown Docks",
							City:   "Rockwall",
							Planet: "Persphone",
							Phone:  "none",
						},
					},
				},
			}
			if user, _, err := handles.Request[User](suite.handler, &create); err == nil {
				suite.Greater(user.Id, 0)
			} else {
				suite.Fail("unexpected error", err.Error())
			}
		})

		suite.Run("Invalid ForSource", func() {
			create := CreateUser{
				User{
					Age:            200,
					Email:          "Badger.Smith",
					FavouriteColor: "#000-",
					Addresses: []Address{
						{},
					},
				},
			}
			if _, _, err := handles.Request[User](suite.handler, &create); err != nil {
				suite.IsType(&validates.Outcome{}, err)
				outcome := err.(*validates.Outcome)
				suite.False(outcome.Valid())
				user := outcome.Path("User")
				suite.Equal("Addresses: (0: (City: Key: 'CreateUser.User.Addresses[0].City' Error:Field validation for 'City' failed on the 'required' tag; Phone: Key: 'CreateUser.User.Addresses[0].Phone' Error:Field validation for 'Phone' failed on the 'required' tag; Planet: Key: 'CreateUser.User.Addresses[0].Planet' Error:Field validation for 'Planet' failed on the 'required' tag; Street: Key: 'CreateUser.User.Addresses[0].Street' Error:Field validation for 'Street' failed on the 'required' tag)); Age: Key: 'CreateUser.User.Age' Error:Field validation for 'Age' failed on the 'lte' tag; Email: Key: 'CreateUser.User.Email' Error:Field validation for 'Email' failed on the 'email' tag; FavouriteColor: Key: 'CreateUser.User.FavouriteColor' Error:Field validation for 'FavouriteColor' failed on the 'iscolor' tag; FirstName: Key: 'CreateUser.User.FirstName' Error:Field validation for 'FirstName' failed on the 'required' tag; LastName: Key: 'CreateUser.User.LastName' Error:Field validation for 'LastName' failed on the 'required' tag", user.Error())
			} else {
				suite.Fail("expected error")
			}
		})
	})

	suite.Run("No Tags", func() {
		suite.Run("Valid ForSource", func() {
			create := CreateUserNoTags{
				UserNoTags{
					FirstName:      "Badger",
					LastName:       "Smith",
					Age:            52,
					Email:          "Badger.Smith@gmail.com",
					FavouriteColor: "#000",
					Addresses: []AddressNoTags{
						{
							Street: "Eavesdown Docks",
							City:   "Rockwall",
							Planet: "Persphone",
							Phone:  "none",
						},
					},
				},
			}
			if user, _, err := handles.Request[UserNoTags](suite.handler, &create); err == nil {
				suite.Greater(user.Id, 0)
			} else {
				suite.Fail("unexpected error", err.Error())
			}
		})

		suite.Run("Invalid ForSource", func() {
			create := CreateUserNoTags{
				UserNoTags{
					Age:            200,
					Email:          "Badger.Smith",
					FavouriteColor: "#000-",
					Addresses: []AddressNoTags{
						{},
					},
				},
			}
			if _, _, err := handles.Request[UserNoTags](suite.handler, &create); err != nil {
				suite.IsType(&validates.Outcome{}, err)
				outcome := err.(*validates.Outcome)
				suite.False(outcome.Valid())
				user := outcome.Path("User")
				suite.Equal("Addresses: (0: (City: Key: 'CreateUserNoTags.User.Addresses[0].City' Error:Field validation for 'City' failed on the 'required' tag; Phone: Key: 'CreateUserNoTags.User.Addresses[0].Phone' Error:Field validation for 'Phone' failed on the 'required' tag; Planet: Key: 'CreateUserNoTags.User.Addresses[0].Planet' Error:Field validation for 'Planet' failed on the 'required' tag; Street: Key: 'CreateUserNoTags.User.Addresses[0].Street' Error:Field validation for 'Street' failed on the 'required' tag)); Age: Key: 'CreateUserNoTags.User.Age' Error:Field validation for 'Age' failed on the 'lte' tag; Email: Key: 'CreateUserNoTags.User.Email' Error:Field validation for 'Email' failed on the 'email' tag; FavouriteColor: Key: 'CreateUserNoTags.User.FavouriteColor' Error:Field validation for 'FavouriteColor' failed on the 'iscolor' tag; FirstName: Key: 'CreateUserNoTags.User.FirstName' Error:Field validation for 'FirstName' failed on the 'required' tag; LastName: Key: 'CreateUserNoTags.User.LastName' Error:Field validation for 'LastName' failed on the 'required' tag", user.Error())
			} else {
				suite.Fail("expected error")
			}
		})
	})
}

func TestValidateTestSuite(t *testing.T) {
	suite.Run(t, new(ValidatorTestSuite))
}
