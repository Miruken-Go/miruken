// Code generated by https://github.com/Miruken-Go/miruken/tools/cmd/miruken; DO NOT EDIT.

package test

import (
	"github.com/miruken-go/miruken/setup"
)

var TestFeature setup.Feature = setup.FeatureFunc(func(setup *setup.Builder) error {
	setup.Specs(
		&CreateUserIntegrity{},
		&UserHandler{},
	)
	return nil
})
