package test

import (
	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
)

// KeyFactory

type KeyFactory struct {}

func (f *KeyFactory) Create(
	_*struct {
	    miruken.Creates
		foo miruken.Creates `key:"foo"`
	  },
) *Foo {
	return &Foo{Counted{1}}
}

// MultiKeyFactory

type MultiKeyFactory struct {}

func (f *MultiKeyFactory) Create(
	_*struct {
		foo miruken.Creates `key:"foo"`
		bar miruken.Creates `key:"bar"`
	  }, create *miruken.Creates,
) any {
	switch create.Key() {
	case "foo":
		return &Foo{Counted{1}}
	case "bar":
		return &Bar{Counted{2}}
	default:
		return nil
	}
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
			miruken.HandlerSpecs(&KeyFactory{}))
		foo, _, err := miruken.CreateKey[*Foo](handler, "foo")
		suite.Nil(err)
		suite.Equal(1, foo.Count())
	})

	suite.Run("InferKey", func () {
		handler, _ := suite.SetupWith(
			miruken.HandlerSpecs(&KeyFactory{}))
		foo, _, err := miruken.Create[*Foo](handler)
		suite.Nil(err)
		suite.Equal(1, foo.Count())
	})

	suite.Run("MultipleKeys", func() {
		handler, _ := suite.SetupWith(
			miruken.HandlerSpecs(&MultiKeyFactory{}))
		foo, _, err := miruken.CreateKey[any](handler, "foo")
		suite.Nil(err)
		suite.Equal(Foo{Counted{1}}, *foo.(*Foo))
		bar, _, err := miruken.CreateKey[any](handler, "bar")
		suite.Nil(err)
		suite.Equal(Bar{Counted{2}}, *bar.(*Bar))
		_, _, err = miruken.CreateKey[any](handler, "boo")
		suite.NotNil(err)
	})
}

func TestCreatesTestSuite(t *testing.T) {
	suite.Run(t, new(CreatesTestSuite))
}
