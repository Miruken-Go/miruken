package test

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

type (
	Worker interface {
		Work() string
	}

	ApiClient struct {
		Name string
	}

	ApiClientRetry struct {
		*ApiClient
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
		client *ApiClientRetry
	}

	ApiCluster struct {
		workers []Worker
	}

	ApiProvider struct {
		def     ApiClient
		client1 ApiClient
		client2 ApiClient
		client3 ApiClient
	}
)

// ApiClientRetry

func (s *ApiClientRetry) Constructor(
	client *ApiClient,
) {
	s.ApiClient = client
}

// ApiService

func (s *ApiService) Constructor(
	client *ApiClient,
) {
	s.client = client
}

func (s *ApiService) Work() string {
	return s.client.Name + " for ApiService"
}

// ApiService1

func (s *ApiService1) Constructor(client *ApiClient) {
	s.client = client
}

func (s *ApiService1) Work() string {
	return s.client.Name + " for ApiService1"
}

// ApiService2

func (s *ApiService2) Constructor(client *ApiClient) {
	s.client = client
}

func (s *ApiService2) Work() string {
	return s.client.Name + " for ApiService2"
}

// ApiService3

func (s *ApiService3) Work() string {
	return s.client.Name + " for ApiService3"
}

// ApiCluster

func (c *ApiCluster) Constructor(workers []Worker) {
	c.workers = workers
}

func (c *ApiCluster) Work() string {
	if len(c.workers) == 0 {
		return ""
	}
	next := rand.Intn(len(c.workers))
	return "Cluster: " + c.workers[next].Work()
}

// ApiProvider

func (p *ApiProvider) Constructor() {
	p.def = ApiClient{"default"}
	p.client1 = ApiClient{"Client1"}
	p.client2 = ApiClient{"Client2"}
	p.client3 = ApiClient{"Client3"}
}

func (p *ApiProvider) ClientForService1(
	_ *struct {
		provides.It
		provides.For[ApiService1]
	},
) *ApiClient {
	return &p.client1
}

func (p *ApiProvider) ClientForService2(
	_ *struct {
		provides.It
		provides.For[ApiService2]
	},
) *ApiClient {
	return &p.client2
}

func (p *ApiProvider) ClientForService3(
	_ *struct {
		provides.It
		provides.ForGraph[ApiService3]
	},
) *ApiClient {
	return &p.client3
}

// Methods in GO are sorted in lexicographic order and registered
// in that order, so we prefix with Z to ensure it is last.
func (p *ApiProvider) ZDefaultClient(
	_ *provides.It,
) *ApiClient {
	return &p.def
}

func (p *ApiProvider) ApiService3(
	_ *provides.It, client *ApiClientRetry,
) *ApiService3 {
	return &ApiService3{client: client}
}

type ForTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *ForTestSuite) SetupTest() {
	suite.specs = []any{
		&ApiClientRetry{},
		&ApiService{},
		&ApiService1{},
		&ApiService2{},
		&ApiCluster{},
		&ApiProvider{},
	}
}

func (suite *ForTestSuite) Setup(specs ...any) (*context.Context, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return setup.New().Specs(specs...).Context()
}

func (suite *ForTestSuite) TestFor() {
	suite.Run("Default", func() {
		handler, _ := suite.Setup()
		svc1, _, ok, err := miruken.Resolve[*ApiService](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.NotNil(svc1)
		suite.Equal("default for ApiService", svc1.Work())
	})

	suite.Run("Constructor", func() {
		handler, _ := suite.Setup()
		svc1, _, ok, err := miruken.Resolve[*ApiService1](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.NotNil(svc1)
		suite.Equal("Client1 for ApiService1", svc1.Work())

		svc2, _, ok, err := miruken.Resolve[*ApiService2](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.NotNil(svc2)
		suite.Equal("Client2 for ApiService2", svc2.Work())
	})

	suite.Run("Method", func() {
		handler, _ := suite.Setup()
		svc3, _, ok, err := miruken.Resolve[*ApiService3](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.NotNil(svc3)
		suite.Equal("Client3 for ApiService3", svc3.Work())
	})

	suite.Run("Interface", func() {
		handler, _ := suite.Setup(&ApiService2{}, &ApiProvider{})
		wkr, _, ok, err := miruken.Resolve[Worker](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.NotNil(wkr)
		suite.Equal("Client2 for ApiService2", wkr.Work())
	})

	suite.Run("Hierarchy", func() {
		handler, _ := suite.Setup()
		cluster, _, ok, err := miruken.Resolve[*ApiCluster](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.NotNil(cluster)
		work := cluster.Work()
		suite.True(strings.HasPrefix(work, "Cluster: "))
		if strings.HasSuffix(work, "ApiService") {
			suite.Equal("Cluster: default for ApiService", work)
		} else if strings.Contains(work, "ApiService1") {
			suite.Equal("Cluster: Client1 for ApiService1", work)
		} else if strings.Contains(work, "ApiService2") {
			suite.Equal("Cluster: Client2 for ApiService2", work)
		} else if strings.Contains(work, "ApiService3") {
			suite.Equal("Cluster: Client3 for ApiService3", work)
		}
	})
}

func TestForTestSuite(t *testing.T) {
	suite.Run(t, new(ForTestSuite))
}
