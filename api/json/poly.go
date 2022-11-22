package json

import (
	"fmt"
	"github.com/miruken-go/miruken"
)

type (
	TypeFieldHandling uint8

	TypeProfileMapper struct {}
)

const (
	TypeFieldHandlingNone TypeFieldHandling = 0
	TypeFieldHandlingRoot TypeFieldHandling = 1 << iota
)

// TypeProfileMapper

func (m *TypeProfileMapper) DotNet(
	_*struct{
		miruken.Maps   `key:"dotnet"`
		miruken.Format `as:"json:type"`
	  }, _ *miruken.Maps,
) string {
	return "$type"
}

func (m *TypeProfileMapper) Java(
	_*struct{
		miruken.Maps   `key:"java"`
		miruken.Format `as:"json:type"`
	  }, _ *miruken.Maps,
) string {
	return "@type"
}

func (m *TypeProfileMapper) Default(
	_*struct{
		miruken.Maps
		miruken.Format `as:"json:type"`
	  }, _ *miruken.Maps,
) string {
	return "@type"
}

// RegisterTypeProfile registers a new type profile with
// the name of the json field used to identify the local
// type in polymorphic content.
func RegisterTypeProfile(profile string, field string) {
	if len(profile) == 0 {
		panic("profile cannot be empty")
	}
	if len(field) == 0 {
		panic("field cannot be empty")
	}
	if _, ok := typeProfiles[profile]; ok {
		panic(fmt.Errorf("json: type profile \"%s\" exists", profile))
	}
}

// TypeProfileFieldName returns the field name associated with
// the type profile for interpreting polymorphic json content.
// If profile is not found, the default profile is used.
func TypeProfileFieldName(profile string) (string, string) {
	if len(profile) == 0 {
		profile = _defaultTypeProfile
	}
	if field, ok := typeProfiles[profile]; !ok {
		profile = _defaultTypeProfile
		return typeProfiles[profile], profile
	} else {
		return field, profile
	}
}

// DefaultTypeProfile gets the default profile used to
// interpret polymorphic json content.
func DefaultTypeProfile() string {
	return _defaultTypeProfile
}

// SetDefaultTypeProfile sets the default profile used to
// interpret polymorphic json content.
func SetDefaultTypeProfile(profile string) {
	if len(profile) == 0 {
		panic("profile cannot be empty")
	}
	if _, ok := typeProfiles[profile]; !ok {
		panic(fmt.Errorf("json: type profile \"%s\" not found", profile))
	}
	_defaultTypeProfile = profile
}

var typeProfiles = map[string]string{
	"dotnet": "$type",
	"java": "@type",
	"go": "@type",
}

var _defaultTypeProfile = "go"