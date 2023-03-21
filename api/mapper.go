package api

import (
	"bytes"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/constraints"
	"github.com/miruken-go/miruken/maps"
)

// Mapper provides common mapping type conversions.
type Mapper struct{}

func (m *Mapper) ToString(
	_*struct{
		maps.Format `to:"*"`
	  }, it *maps.It,
	ctx miruken.HandleContext,
) (string, error) {
	format, _ := constraints.First[*maps.Format](it)
	b, bp, err := maps.Map[[]byte](ctx.Composer(), it.Source(), format)
	if err != nil {
		return "", err
	}
	if bp != nil {
		if b, err = bp.Await(); err != nil {
			return "", err
		}
	}
	return string(b), nil
}

func (m *Mapper) ToBytesBuffer(
	_*struct{
		maps.Format `to:"*"`
	  }, it *maps.It,
	ctx miruken.HandleContext,
) (*bytes.Buffer, error) {
	format, _ := constraints.First[*maps.Format](it)
	b, bp, err := maps.Map[[]byte](ctx.Composer(), it.Source(), format)
	if err != nil {
		return nil, err
	}
	if bp != nil {
		if b, err = bp.Await(); err != nil {
			return nil, err
		}
	}
	return bytes.NewBuffer(b), nil
}
