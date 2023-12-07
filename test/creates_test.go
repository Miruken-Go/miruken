package test

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
)

// KeyFactory

type KeyFactory struct {}

func (f *KeyFactory) Create(
	_*struct{
	    creates.It
		_ creates.It `key:"foo"`
	  },
) *Foo {
	return &Foo{Counted{1}}
}

// MultiKeyFactory

type MultiKeyFactory struct {
	foo Foo
	bar Bar
}

func (f *MultiKeyFactory) Create(
	_*struct{
		provides.Single
		_ creates.It  `key:"foo"`
		_ creates.It  `key:"bar"`
		_ provides.It `key:"foo"`
		_ provides.It `key:"bar"`
	  },
	c *creates.It,
	p *provides.It,
) any {
	if c != nil {
		switch c.Key() {
		case "foo":
			f.foo.Inc()
			return f.foo
		case "bar":
			f.bar.Inc()
			return f.bar
		}
	} else if p != nil {
		switch p.Key() {
		case "foo":
			f.foo.Inc()
			return f.foo
		case "bar":
			f.bar.Inc()
			return f.bar
		}
	}
	return nil
}

type CreatesTestSuite struct {
	suite.Suite
}

func (suite *CreatesTestSuite) Setup() (miruken.Handler, error) {
	return setup.New(TestFeature).ExcludeSpecs(
		func (spec miruken.HandlerSpec) bool {
			switch ts := spec.(type) {
			case miruken.TypeSpec:
				return strings.Contains(ts.Name(), "Invalid")
			default:
				return false
			}
		}).Context()
}

func (suite *CreatesTestSuite) TestCreates() {
	suite.Run("Invariant", func() {
		handler, _ := setup.New().
			Specs(&SpecificationProvider{}).
			Handlers(&SpecificationProvider{foo: Foo{Counted{10}}}).
			Context()
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
		handler, _ := setup.New().
			Specs(&SimpleAsyncProvider{}, &ComplexAsyncProvider{}).
			Context()
		c, pc, err := miruken.Create[*ComplexAsyncProvider](handler)
		suite.Nil(err)
		suite.Nil(c)
		suite.NotNil(pc)
		c, err = pc.Await()
		suite.Nil(err)
		suite.NotNil(c)
		suite.Equal(2, c.bar.Count())
	})

	suite.Run("Key", func () {
		handler, _ := setup.New().
			Specs(&KeyFactory{}).
			Context()
		foo, _, err := miruken.CreateKey[*Foo](handler, "foo")
		suite.Nil(err)
		suite.Equal(1, foo.Count())
	})

	suite.Run("InferKey", func () {
		handler, _ := setup.New().
			Specs(&KeyFactory{}).
			Context()
		foo, _, err := miruken.Create[*Foo](handler)
		suite.Nil(err)
		suite.Equal(1, foo.Count())
	})

	suite.Run("MultipleKeys", func() {
		handler, _ := setup.New().
			Specs(&MultiKeyFactory{}).
			Context()
		foo, _, err := miruken.CreateKey[Foo](handler, "foo")
		suite.Nil(err)
		suite.Equal(Foo{Counted{1}}, foo)
		foo, _, err = miruken.CreateKey[Foo](handler, "foo")
		suite.Nil(err)
		suite.Equal(Foo{Counted{2}}, foo)
		bar, _, err := miruken.CreateKey[Bar](handler, "bar")
		suite.Nil(err)
		suite.Equal(Bar{Counted{1}}, bar)
		bar, _, err = miruken.CreateKey[Bar](handler, "bar")
		suite.Nil(err)
		suite.Equal(Bar{Counted{2}}, bar)
		_, _, err = miruken.CreateKey[any](handler, "boo")
		suite.NotNil(err)
	})

	suite.Run("MultipleKeyCallbacks", func() {
		handler, _ := setup.New().
			Specs(&MultiKeyFactory{}).
			Context()
		foo, _, err := miruken.CreateKey[Foo](handler, "foo")
		suite.Nil(err)
		suite.Equal(Foo{Counted{1}}, foo)
		foo, _, err = miruken.CreateKey[Foo](handler, "foo")
		suite.Nil(err)
		suite.Equal(Foo{Counted{2}}, foo)
		foo, _, _, err = miruken.ResolveKey[Foo](handler, "foo")
		suite.Nil(err)
		suite.Equal(Foo{Counted{3}}, foo)
		foo, _, _, err = miruken.ResolveKey[Foo](handler, "foo")
		suite.Nil(err)
		suite.Equal(Foo{Counted{3}}, foo)
	})
}

func TestCreatesTestSuite(t *testing.T) {
	suite.Run(t, new(CreatesTestSuite))
}
