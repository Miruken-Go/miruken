package test

import (
	"github.com/miruken-go/miruken/security"
	"github.com/stretchr/testify/suite"
	"testing"
)

type (
	UserId struct {
		Id int32
	}

	Password struct {
		Text string
	}
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
		ps := []any{UserId{2}, security.Group{Name: "Users"},
			security.Role{Name: "Billing"}}
		sub := security.NewSubject(security.Principals(ps...))
		suite.NotNil(sub)
		suite.ElementsMatch(sub.Principals(), ps)
	})

	suite.Run("AddPrincipals", func() {
		ps := []any{UserId{2}, security.Group{Name: "Users"},
			security.Role{Name: "Billing"}}
		sub := security.NewSubject()
		sub.AddPrincipals(ps...)
		suite.ElementsMatch(sub.Principals(), ps)
	})

	suite.Run("DistinctPrincipals", func() {
		ps := []any{UserId{2}, security.Group{Name: "Users"},
			security.Role{Name: "Billing"}}
		sub := security.NewSubject(security.Principals(ps...))
		suite.Len(sub.Principals(), 3)
		suite.ElementsMatch(sub.Principals(), ps)
		sub.AddPrincipals(ps...)
		suite.Len(sub.Principals(), 3)
		suite.ElementsMatch(sub.Principals(), ps)
	})

	suite.Run("NewWithCredentials", func() {
		sub := security.NewSubject(security.Credentials(Password{"1234"}))
		suite.NotNil(sub)
		suite.ElementsMatch(sub.Credentials(), []any{Password{"1234"}})
	})

	suite.Run("AddCredentials", func() {
		sub := security.NewSubject()
		sub.AddCredentials(Password{"1234"})
		suite.ElementsMatch(sub.Credentials(), []any{Password{"1234"}})
	})

	suite.Run("DistinctCredentials", func() {
		sub := security.NewSubject(security.Credentials(Password{"1234"}))
		suite.NotNil(sub)
		suite.Len(sub.Credentials(), 1)
		suite.ElementsMatch(sub.Credentials(), []any{Password{"1234"}})
		sub.AddCredentials(Password{"1234"})
		suite.Len(sub.Credentials(), 1)
		suite.ElementsMatch(sub.Credentials(), []any{Password{"1234"}})
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
			sys.AddPrincipals(security.Role{Name: "Admin"})
		})

		suite.Run("ImmutableCredentials", func() {
			defer func() {
				if r := recover(); r != nil {
					suite.Equal("system subject is immutable", r)
				}
			}()
			sys := security.SystemSubject
			sys.AddCredentials(Password{"1234"})
		})
	})
}

func TestSubjectTestSuite(t *testing.T) {
	suite.Run(t, new(SubjectTestSuite))
}
