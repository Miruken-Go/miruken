package test

import (
	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
)

type AliasFactory struct {}

func (f *AliasFactory) Create(
	_*struct {
		account miruken.Creates `key:"customer.Account"`
		profile miruken.Creates `key:"customer.Profile"`
	  },
) any {
	return nil
}

type CreatesTestSuite struct {
	suite.Suite
}

func (suite *CreatesTestSuite) Setup() (miruken.Handler, error) {
	return miruken.Setup(TestFeature, miruken.ExcludeHandlerSpecs(
		func (spec miruken.HandlerSpec) bool {
			switch ts := spec.(type) {
			case miruken.HandlerTypeSpec:
				return strings.Contains(ts.Name(), "Invalid")
			default:
				return false
			}
		}))
}

func (suite *CreatesTestSuite) SetupWith(
	features ... miruken.Feature,
) (miruken.Handler, error) {
	return miruken.Setup(features...)
}

func (suite *CreatesTestSuite) TestCreates() {
	suite.Run("Invariant", func() {
		handler, _ := suite.SetupWith(
			miruken.HandlerSpecs(&SpecificationProvider{}),
			miruken.Handlers(&SpecificationProvider{foo: Foo{Counted{10}}}))
		foo, _, err := miruken.Create[*Foo](handler)
		suite.Nil(err)
		suite.Equal(11, foo.Count())
	})

	suite.Run("Infer", func() {
		handler, _ := suite.Setup()
		multiProvider, _, err := miruken.Create[*MultiProvider](handler)
		suite.NotNil(multiProvider)
		suite.Nil(err)
	})

	suite.Run("Returns Promise", func() {
		handler, _ := suite.SetupWith(
			miruken.HandlerSpecs(&SimpleAsyncProvider{}),
			miruken.HandlerSpecs(&ComplexAsyncProvider{}))
		c, pc, err := miruken.Create[*ComplexAsyncProvider](handler)
		suite.Nil(err)
		suite.Nil(c)
		suite.NotNil(pc)
		c, err = pc.Await()
		suite.Nil(err)
		suite.NotNil(c)
		suite.Equal(1, c.bar.Count())
	})

	suite.Run("Key", func () {
		handler, _ := suite.SetupWith(
			miruken.HandlerSpecs(&KeyProvider{}))
		foo, _, err := miruken.CreateKey[*Foo](handler, "Foo")
		suite.Nil(err)
		suite.Equal(1, foo.Count())
	})

	suite.Run("MultipleKeys", func() {
		handler, _ := suite.SetupWith(
			miruken.HandlerSpecs(&AliasFactory{}))
		account, _, err := miruken.CreateKey[string](handler, "customer.Account")
		suite.Nil(err)
		suite.Equal("account", account)
	})
}

func TestCreatesTestSuite(t *testing.T) {
	suite.Run(t, new(CreatesTestSuite))
}
