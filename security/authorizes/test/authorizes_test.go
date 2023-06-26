package test

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/constraints"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/security"
	"github.com/miruken-go/miruken/security/authorizes"
	"github.com/miruken-go/miruken/security/principal"
	"github.com/stretchr/testify/suite"
	"testing"
)

//go:generate $GOPATH/bin/miruken -tests

type (
	Money uint64

	TransferFunds struct {
		Amount Money
	}

	TransferFundsAccessPolicy struct {}

	Account struct {
		Balance Money
	}
)


// TransferFundsAccessPolicy

func (t *TransferFundsAccessPolicy) AuthorizeTransfer(
	_ *authorizes.It, transfer TransferFunds,
	subject security.Subject,
) bool {
	if amount := transfer.Amount; amount < 10000 {
		return true
	}
	return principal.All(subject, principal.Role("manager"))
}

func (t *TransferFundsAccessPolicy) AuthorizeTransferFast(
	_*struct{
		authorizes.It
		constraints.Named `name:"fast"`
	  }, transfer TransferFunds,
	subject security.Subject,
) *promise.Promise[bool] {
	return promise.Resolve(transfer.Amount < 1000 &&
		principal.All(subject, principal.Role("owner")))
}

// Account

func (a *Account) Transfer(
	_*struct {
		handles.It
		authorizes.Required
	  }, transfer TransferFunds,
) Money {
	a.Balance += transfer.Amount
	return a.Balance
}


type AuthorizesTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *AuthorizesTestSuite) SetupTest() {
	suite.specs = []any{
		&Account{},
		&TransferFundsAccessPolicy{},
	}
}

func (suite *AuthorizesTestSuite) Setup(specs ...any) (miruken.Handler, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return miruken.Setup().Specs(specs...).Handler()
}

func (suite *AuthorizesTestSuite) TestAuthorizes() {
	suite.Run("Authorize", func () {
		suite.Run("Default", func() {
			handler, _ := miruken.Setup().Handler()
			transfer := TransferFunds{Amount: 1000}
			handler = miruken.BuildUp(handler, provides.With(security.NewSubject()))
			grant, _, err := authorizes.Access(handler, transfer)
			suite.Nil(err)
			suite.True(grant)
		})

		suite.Run("RequiresPolicy", func() {
			handler, _ := miruken.Setup().Handler()
			handler = miruken.BuildUp(handler,
				provides.With(security.NewSubject()),
				miruken.Options(authorizes.Options{RequirePolicy: true}))
			transfer := TransferFunds{Amount: 1000}
			grant, _, err := authorizes.Access(handler, transfer)
			suite.Nil(err)
			suite.False(grant)
		})

		suite.Run("Granted", func() {
			handler, _ := suite.Setup()
			transfer := TransferFunds{Amount: 1000}
			handler = miruken.BuildUp(handler, provides.With(security.NewSubject()))
			grant, _, err := authorizes.Access(handler, transfer)
			suite.Nil(err)
			suite.True(grant)
		})

		suite.Run("DeniedWithoutRole", func() {
			handler, _ := suite.Setup()
			transfer := TransferFunds{Amount: 1000000}
			handler = miruken.BuildUp(handler, provides.With(security.NewSubject()))
			grant, _, err := authorizes.Access(handler, transfer)
			suite.Nil(err)
			suite.False(grant)
		})

		suite.Run("GrantedWithRole", func() {
			handler, _ := suite.Setup()
			transfer := TransferFunds{Amount: 1000000}
			subject  := security.NewSubject(
				security.WithPrincipals(principal.Role("manager")))
			handler = miruken.BuildUp(handler, provides.With(subject))
			grant, _, err := authorizes.Access(handler, transfer)
			suite.Nil(err)
			suite.True(grant)
		})

		suite.Run("ConstrainedPolicy", func() {
			handler, _ := suite.Setup()
			transfer := TransferFunds{Amount: 500}
			subject  := security.NewSubject()
			handler = miruken.BuildUp(handler, provides.With(subject))
			g, gp, err := authorizes.Access(handler, transfer, "fast")
			suite.Nil(err)
			suite.NotNil(gp)
			g, err = gp.Await()
			suite.False(g)
			suite.Nil(err)
			subject.AddPrincipals(principal.Role("owner"))
			g, gp, err = authorizes.Access(handler, transfer, "fast")
			suite.Nil(err)
			suite.NotNil(gp)
			g, err = gp.Await()
			suite.True(g)
		})

		suite.Run("Filter", func() {
			suite.Run("NotHandledWithoutSubject", func() {
				handler, _ := suite.Setup()
				transfer := TransferFunds{Amount: 1000}
				_, _, err := handles.Request[int](handler, transfer)
				suite.IsType(err, &miruken.NotHandledError{})
			})

			suite.Run("Granted", func() {
				handler, _ := suite.Setup()
				transfer := TransferFunds{Amount: 1000}
				handler = miruken.BuildUp(handler, provides.With(security.NewSubject()))
				balance, _, err := handles.Request[int](handler, transfer)
				suite.Nil(err)
				suite.Equal(1000, balance)
			})

			suite.Run("DeniedWithoutRole", func() {
				handler, _ := suite.Setup()
				transfer := TransferFunds{Amount: 20000}
				subject  := security.NewSubject(
					security.WithPrincipals(principal.Role("manager")))
				handler = miruken.BuildUp(handler, provides.With(subject))
				balance, _, err := handles.Request[int](handler, transfer)
				suite.Nil(err)
				suite.Equal(20000, balance)
			})
		})
	})
}

func TestAuthorizesTestSuite(t *testing.T) {
	suite.Run(t, new(AuthorizesTestSuite))
}
