package test

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/suite"
	"testing"
)

type (
	Mailer interface {
		SendMail(to string, msg string)
	}

	MailerStub struct {}

	SendMail struct {
		miruken.LateSideEffect
		To  string
		Msg string
	}

	CreateAccount struct {
		Name string
		Email string
	}
	AccountHandler struct {}
)


func (m *MailerStub) SendMail(to string, msg string) {
	fmt.Println("Sending", msg, "to", to)
}


func (s SendMail) LateApply(
	mailer Mailer,
)  ([]any, *promise.Promise[[]any], error) {
	mailer.SendMail(s.To, s.Msg)
	return nil, nil, nil
}


func (a *AccountHandler) CreateAccount(
	_ *handles.It, create CreateAccount,
) SendMail {
	msg := fmt.Sprintf("Welcome %s", create.Name)
	return SendMail{To: create.Email, Msg: msg}
}


type SideEffectTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *SideEffectTestSuite) SetupTest() {
	suite.specs =  []any{
		&MailerStub{},
		&AccountHandler{},
	}
}

func (suite *SideEffectTestSuite) Setup(specs ...any) (miruken.Handler, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return miruken.Setup().Specs(specs...).Handler()
}

func (suite *SideEffectTestSuite) TestSideEffects() {
	suite.Run("Side Effects", func () {
		handler, _ := suite.Setup()
		create := CreateAccount{"John Doe", "jd@gmail.com"}
		result := handler.Handle(create, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
	})
}

func TestSideEffectsTestSuite(t *testing.T) {
	suite.Run(t, new(SideEffectTestSuite))
}
