package test

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/constraints"
	"github.com/miruken-go/miruken/provides"
	"github.com/stretchr/testify/suite"
	"testing"
)

type (
	Worker interface {
		Work() string
	}

	ApiClient struct {
		Name string
	}

	ApiService struct {
		client *ApiClient
	}

	ApiService1 struct {
		client *ApiClient
	}

	ApiService2 struct {
		client *ApiClient
	}

	ApiService3 struct {
		client *ApiClient
	}

	ApiProvider struct {
		def     ApiClient
		client1 ApiClient
		client2 ApiClient
	}
)


// ApiService

func (s *ApiService) Constructor(
	client *ApiClient,
) {
	s.client = client
}

func (s *ApiService) Work() string {
	return s.client.Name + " for ApiService"
}


func (s *ApiService1) Constructor(
	_*struct{constraints.ForMe}, client *ApiClient,
) {
	s.client = client
}

func (s *ApiService1) Work() string {
	return s.client.Name + " for ApiService1"
}


// ApiService2

func (s *ApiService2) Constructor(
	_*struct{constraints.ForMe}, client *ApiClient,
) {
	s.client = client
}

func (s *ApiService2) Work() string {
	return s.client.Name + " for ApiService2"
}


// ApiService3

func (s *ApiService3) Work() string {
	return s.client.Name + " for ApiService3"
}


// ApiProvider

func (p *ApiProvider) Constructor() {
	p.def     = ApiClient{"default"}
	p.client1 = ApiClient{"Client1"}
	p.client2 = ApiClient{"Client2"}
}

func (p *ApiProvider) DefaultClient(
	_ *provides.It,
) *ApiClient {
	return &p.def
}


func (p *ApiProvider) ClientForService1(
	_*struct{
		provides.It
		constraints.For[ApiService1]
	  },
) *ApiClient {
	return &p.client1
}

func (p *ApiProvider) ClientForService2(
	_*struct{
		provides.It
		constraints.For[ApiService2]
	  },
) *ApiClient {
	return &p.client2
}

func (p *ApiProvider) ClientForService3(
	_*struct{
		provides.It
		constraints.For[ApiService3]
	  },
) *ApiClient {
	return &p.def
}

func (p *ApiProvider) ApiService3(
	_ *provides.It,
	_*struct{constraints.ForMe}, client *ApiClient,
) *ApiService3 {
	return &ApiService3{client: client}
}


type ForTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *ForTestSuite) SetupTest() {
	suite.specs = []any{
		&ApiService{},
		&ApiService1{},
		&ApiService2{},
		&ApiProvider{},
	}
}

func (suite *ForTestSuite) Setup(specs...any) (miruken.Handler, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return miruken.Setup().Specs(specs...).Handler()
}

func (suite *ForTestSuite) TestFrom() {
	suite.Run("Default", func () {
		handler, _ := suite.Setup()
		svc1, _, err := miruken.Resolve[*ApiService](handler)
		suite.Nil(err)
		suite.NotNil(svc1)
		suite.Equal("default for ApiService", svc1.Work())
	})

	suite.Run("Constructor", func () {
		handler, _ := suite.Setup()
		svc1, _, err := miruken.Resolve[*ApiService1](handler)
		suite.Nil(err)
		suite.NotNil(svc1)
		suite.Equal("Client1 for ApiService1", svc1.Work())

		svc2, _, err := miruken.Resolve[*ApiService2](handler)
		suite.Nil(err)
		suite.NotNil(svc2)
		suite.Equal("Client2 for ApiService2", svc2.Work())
	})

	suite.Run("Method", func () {
		handler, _ := suite.Setup()
		svc3, _, err := miruken.Resolve[*ApiService3](handler)
		suite.Nil(err)
		suite.NotNil(svc3)
		suite.Equal("default for ApiService3", svc3.Work())
	})

	suite.Run("Interface", func () {
		handler, _ := suite.Setup(&ApiService2{}, &ApiProvider{})
		wkr, _, err := miruken.Resolve[Worker](handler)
		suite.Nil(err)
		suite.NotNil(wkr)
		suite.Equal("Client2 for ApiService2", wkr.Work())
	})
}

func TestForTestSuite(t *testing.T) {
	suite.Run(t, new(ForTestSuite))
}