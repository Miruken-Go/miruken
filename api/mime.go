package api

import "github.com/miruken-go/miruken/maps"

var (
	ToJson   = maps.To("application/json")
	FromJson = maps.From("application/json")
)
