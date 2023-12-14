package validates

import (
	"reflect"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/promise"
)

type (
	// Required is a FilterProvider for validation.
	Required struct {
		validateOutput bool
	}

	// filter validates the current input of the pipeline execution.
	// if validateOutput is true, the output is validated too.
	filter struct{}
)

// Constraints

func (r *Required) InitWithTag(tag reflect.StructTag) error {
	if v, ok := tag.Lookup("validates"); ok {
		r.validateOutput = v == "output"
	}
	return nil
}

func (r *Required) ValidateOutput() bool {
	return r.validateOutput
}

func (r *Required) Required() bool {
	return false
}

func (r *Required) AppliesTo(
	callback miruken.Callback,
) bool {
	h, ok := callback.(*handles.It)
	return ok && !internal.IsNil(h.Source())
}

func (r *Required) Filters(
	binding miruken.Binding,
	callback any,
	composer miruken.Handler,
) ([]miruken.Filter, error) {
	return filters, nil
}

// filter

func (f filter) Order() int {
	return miruken.FilterStageValidation
}

func (f filter) Next(
	self miruken.Filter,
	next miruken.Next,
	ctx miruken.HandleContext,
	provider miruken.FilterProvider,
) (out []any, pout *promise.Promise[[]any], err error) {
	if cp, ok := provider.(*Required); ok {
		callback := ctx.Callback
		composer := ctx.Composer
		outcomeIn, poi, errIn := Constraints(composer, callback.Source())
		if errIn != nil {
			// error validating input
			return nil, nil, errIn
		}
		if poi == nil {
			// if invalid return input results
			if !outcomeIn.Valid() {
				return nil, nil, outcomeIn
			}
			// perform the next step in the pipeline
			if out, pout, err = next.Pipe(); !(err == nil && cp.validateOutput) {
				// if error or skip output validation, return output
				return
			} else if pout == nil {
				// validates output if available
				if len(out) > 0 && !internal.IsNil(out[0]) {
					outcomeOut, poo, errOut := Constraints(composer, out[0])
					if errOut != nil {
						// error validating so return
						return nil, nil, errOut
					}
					if poo == nil {
						// synchronous output validation
						if !outcomeOut.Valid() {
							// invalid so return output results
							return nil, nil, outcomeOut
						}
					} else {
						// asynchronous output validation
						return nil, promise.Then(poo, func(outcome *Outcome) []any {
							// if invalid return output results
							if !outcome.Valid() {
								panic(outcome)
							}
							return out
						}), nil
					}
				}
				return
			} else {
				// asynchronous output validation
				return nil, promise.Then(pout, func(oo []any) []any {
					if len(oo) > 0 && !internal.IsNil(oo[0]) {
						outcomeOut, poo, errOut := Constraints(composer, oo[0])
						if errOut != nil {
							// error validating input
							panic(errOut)
						}
						if poo != nil {
							// resolve output validation results
							if outcomeOut, errOut = poo.Await(); errOut != nil {
								// resolution failed so return
								panic(errOut)
							}
						} else if !outcomeOut.Valid() {
							// invalid so return output results
							panic(outcomeOut)
						}
					}
					return oo
				}), nil
			}
		}
		// asynchronous input validation
		return nil, promise.Then(poi, func(outcome *Outcome) []any {
			// if invalid return input results
			if !outcome.Valid() {
				panic(outcome)
			}
			oo := next.PipeAwait()
			// validates output if requested and available
			if cp.validateOutput && len(oo) > 0 && !internal.IsNil(oo[0]) {
				outcomeOut, poo, errOut := Constraints(composer, oo[0])
				if errOut != nil {
					// error validating output
					panic(errOut)
				}
				if poo != nil {
					// resolve output validation results
					if outcomeOut, errOut = poo.Await(); errOut != nil {
						// resolution failed so return
						panic(errOut)
					}
				} else if !outcomeOut.Valid() {
					// invalid so return output results
					panic(outcomeOut)
				}
			}
			return oo
		}), nil
	}
	return next.Abort()
}

var filters = []miruken.Filter{filter{}}
