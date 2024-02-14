package test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

type (
	Account struct {
		Id    int
		Name  string ``
		Email string
	}

	Database interface {
		NewAccount(
			id int,
			name string,
			email string,
		) (*promise.Promise[*Account], error)
	}

	DatabaseStub struct {
		Accounts map[int]*Account
	}

	NewEntity struct {
		Id    int
		Name  string
		Email string
	}

	Mailer interface {
		SendMail(to string, msg string) error
	}

	MailerStub struct {
		Log map[string]string
	}

	SendMail struct {
		To  string
		Msg string
	}

	PostMsg struct {
		To  string
		Data []byte
	}

	CreateAccount struct {
		Name  string
		Email string
	}

	ConfirmAccount struct {
		Name  string
		Email string
	}

	RecordTransaction struct {
		Activity string
	}

	AccountHandler struct {
		nextId int
	}
)

func (d *DatabaseStub) Constructor() {
	d.Accounts = make(map[int]*Account)
}

func (d *DatabaseStub) NewAccount(
	id    int,
	name  string,
	email string,
) (*promise.Promise[*Account], error) {
	switch name {
	case "Fail":
		return nil, errors.New("database is unavailable")
	case "FailAsync":
		return promise.Reject[*Account](errors.New("database is busy")), nil
	}
	account := &Account{id, name, email}
	d.Accounts[id] = account
	return promise.Resolve(account), nil
}

func (e NewEntity) Apply(
	database Database,
) (promise.Reflect, error) {
	return database.NewAccount(e.Id, e.Name, e.Email)
}

func (m *MailerStub) Constructor() {
	m.Log = make(map[string]string)
}

func (m *MailerStub) SendMail(to string, msg string) error {
	if strings.Contains(msg, "fail") {
		return fmt.Errorf("mail failed: %s", msg)
	}
	m.Log[to] = msg
	return nil
}

func (s SendMail) Apply(
	mailer Mailer,
) error {
	return mailer.SendMail(s.To, s.Msg)
}

func (p PostMsg) Apply(
) (promise.Reflect, error) {
	return promise.New(nil, func(
		resolve func(struct{}), reject func(error), onCancel func(func())) {
		println(string(p.Data))
		resolve(struct{}{})
	}), nil
}

func (a *AccountHandler) CreateAccount(
	_ *handles.It, create CreateAccount,
) (int, NewEntity, SendMail, PostMsg, RecordTransaction, error) {
	a.nextId++
	msg := fmt.Sprintf("Welcome %s", create.Name)
	return a.nextId,
		NewEntity{Id: a.nextId, Name: create.Name, Email: create.Email},
		SendMail{To: create.Email, Msg: msg},
		PostMsg{To: "http://localhost:8080", Data: []byte("Account Created")},
		RecordTransaction{fmt.Sprintf("Account '%s' Created", create.Email) },
		nil
}

func (a *AccountHandler) ConfirmAccount(
	_ *handles.It, confirm ConfirmAccount,
) (*promise.Promise[string], SendMail, PostMsg, RecordTransaction, miruken.HandleResult) {
	msg := fmt.Sprintf("Confirm your account %s", confirm.Name)
	return promise.Resolve(confirm.Email),
		SendMail{To: confirm.Email, Msg: msg},
		PostMsg{To: "http://localhost:8080", Data: []byte("Hello World!")},
		RecordTransaction{fmt.Sprintf("Account '%s' Confirmed", confirm.Email) },
		miruken.Handled
}

func (a *AccountHandler) RecordTransaction(
	_ *handles.It, record RecordTransaction,
) {
	println(record.Activity)
}


type IntentTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *IntentTestSuite) SetupTest() {
	suite.specs = []any{
		&DatabaseStub{},
		&MailerStub{},
		&AccountHandler{},
	}
}

func (suite *IntentTestSuite) Setup(specs ...any) (miruken.Handler, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return setup.New().Specs(specs...).Context()
}

func (suite *IntentTestSuite) TestIntents() {
	suite.Run("Single", func() {
		handler, _ := suite.Setup()
		confirm := ConfirmAccount{"John Doe", "jd@gmail.com"}
		r, pr, err := handles.Request[string](handler, confirm)
		suite.Nil(err)
		suite.NotNil(pr)
		r, err = pr.Await()
		suite.Nil(err)
		suite.Equal("jd@gmail.com", r)
		mailer, _, _, _ := provides.Type[*MailerStub](handler)
		suite.Equal("Confirm your account John Doe", mailer.Log["jd@gmail.com"])
	})

	suite.Run("Multiple", func() {
		handler, _ := suite.Setup()
		create := CreateAccount{"John Doe", "jd@gmail.com"}
		id, pid, err := handles.Request[int](handler, create)
		suite.Nil(err)
		suite.NotNil(pid)
		suite.Empty(id)
		id, err = pid.Await()
		suite.Nil(err)
		suite.NotEmpty(id)
		db, _, _, _ := provides.Type[*DatabaseStub](handler)
		suite.Equal("John Doe", db.Accounts[id].Name)
		suite.NotNil(db)
		mailer, _, _, _ := provides.Type[*MailerStub](handler)
		suite.Equal("Welcome John Doe", mailer.Log["jd@gmail.com"])
	})

	suite.Run("Error", func() {
		handler, _ := suite.Setup()
		create := CreateAccount{"Fail", "fail@gmail.com"}
		_, pid, err := handles.Request[int](handler, create)
		suite.Nil(pid)
		suite.NotNil(err)
		suite.Equal("database is unavailable", err.Error())
	})

	suite.Run("ErrorAsync", func() {
		handler, _ := suite.Setup()
		create := CreateAccount{"FailAsync", "fail@gmail.com"}
		_, pid, err := handles.Request[int](handler, create)
		suite.NotNil(pid)
		suite.Nil(err)
		_, err = pid.Await()
		suite.Equal("database is busy", err.Error())
	})
}

func TestIntentsTestSuite(t *testing.T) {
	suite.Run(t, new(IntentTestSuite))
}
