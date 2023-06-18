package test

import (
	"github.com/miruken-go/miruken/security"
	"github.com/miruken-go/miruken/security/principal"
	"github.com/stretchr/testify/suite"
	"testing"
)

type (
	UserId   int32
	Password string
)

type SubjectTestSuite struct {
	suite.Suite
}

func (suite *SubjectTestSuite) TestSubject() {
	suite.Run("New", func() {
		sub := security.NewSubject()
		suite.NotNil(sub)
	})

	suite.Run("NewWithPrincipals", func() {
		ps := []any{UserId(2), principal.Group("Users"),
			principal.Role("Billing")}
		sub := security.NewSubject(security.WithPrincipals(ps...))
		suite.NotNil(sub)
		suite.ElementsMatch(ps, sub.Principals())
	})

	suite.Run("AddPrincipals", func() {
		ps := []any{UserId(2), principal.Group("Users"),
			principal.Role("Billing")}
		sub := security.NewSubject()
		sub.AddPrincipals(ps...)
		suite.ElementsMatch(ps, sub.Principals())
	})

	suite.Run("DistinctPrincipals", func() {
		ps := []any{UserId(2), principal.Group("Users"),
			principal.Role("Billing")}
		sub := security.NewSubject(security.WithPrincipals(ps...))
		suite.Len(sub.Principals(), 3)
		suite.ElementsMatch(ps, sub.Principals())
		sub.AddPrincipals(ps...)
		suite.Len(sub.Principals(), 3)
		suite.ElementsMatch(ps, sub.Principals())
	})

	suite.Run("RemovePrincipals", func() {
		ps := []any{UserId(2), principal.Group("Users"),
			principal.Role("Billing")}
		sub := security.NewSubject(security.WithPrincipals(ps...))
		sub.RemovePrincipals(principal.Role("Billing"))
		suite.Len(sub.Principals(),2)
		suite.ElementsMatch([]any{UserId(2), principal.Group("Users")}, sub.Principals())
	})

	suite.Run("NewWithCredentials", func() {
		sub := security.NewSubject(security.WithCredentials(Password("1234")))
		suite.NotNil(sub)
		suite.ElementsMatch( []any{Password("1234")}, sub.Credentials())
	})

	suite.Run("AddCredentials", func() {
		sub := security.NewSubject()
		sub.AddCredentials(Password("1234"))
		suite.ElementsMatch( []any{Password("1234")}, sub.Credentials())
	})

	suite.Run("DistinctCredentials", func() {
		sub := security.NewSubject(security.WithCredentials(Password("1234")))
		suite.NotNil(sub)
		suite.Len(sub.Credentials(), 1)
		suite.ElementsMatch([]any{Password("1234")}, sub.Credentials())
		sub.AddCredentials(Password("1234"))
		suite.Len(sub.Credentials(), 1)
		suite.ElementsMatch([]any{Password("1234")}, sub.Credentials())
	})

	suite.Run("RemoveCredentials", func() {
		sub := security.NewSubject(security.WithCredentials(Password("1234")))
		sub.RemoveCredentials(Password("1234"))
		suite.Len(sub.Credentials(),0)
	})

	suite.Run("System", func() {
		suite.Run("New", func() {
			sys := security.SystemSubject
			suite.NotNil(sys)
			suite.Nil(sys.Credentials())
			suite.Equal([]any{security.System}, sys.Principals())
		})

		suite.Run("ImmutablePrincipals", func() {
			defer func() {
				if r := recover(); r != nil {
					suite.Equal("system subject is immutable", r)
				}
			}()
			sys := security.SystemSubject
			sys.AddPrincipals(principal.Role("Admin"))
		})

		suite.Run("ImmutableCredentials", func() {
			defer func() {
				if r := recover(); r != nil {
					suite.Equal("system subject is immutable", r)
				}
			}()
			sys := security.SystemSubject
			sys.AddCredentials(Password("1234"))
		})
	})
}

func TestSubjectTestSuite(t *testing.T) {
	suite.Run(t, new(SubjectTestSuite))
}
