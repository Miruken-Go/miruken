package validate

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

type (
	// Provider is a FilterProvider for validation.
	Provider struct {
		validateOutput bool
	}

	// filter validates the current input of the pipeline execution.
	// if validateOutput is true, the output is validated too.
 	filter struct {}
)


// Provider

func (v *Provider) InitWithTag(tag reflect.StructTag) error {
	if validate, ok := tag.Lookup("validate"); ok {
		v.validateOutput = validate == "output"
	}
	return nil
}

func (v *Provider) Required() bool {
	return false
}

func (v *Provider) AppliesTo(
	callback miruken.Callback,
) bool {
	handles, ok := callback.(*miruken.Handles)
	return ok && !miruken.IsNil(handles.Source())
}

func (v *Provider) Filters(
	binding  miruken.Binding,
	callback any,
	composer miruken.Handler,
) ([]miruken.Filter, error) {
	return _filters, nil
}


// filter

func (f filter) Order() int {
	return miruken.FilterStageValidation
}

func (f filter) Next(
	next     miruken.Next,
	ctx      miruken.HandleContext,
	provider miruken.FilterProvider,
)  (out []any, pout *promise.Promise[[]any], err error) {
	if vp, ok := provider.(*Provider); ok {
		callback := ctx.Callback()
		composer := ctx.Composer()
		outcomeIn, poi, errIn := Validate(composer, callback.Source())
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
			if out, pout, err = next.Pipe(); !(err == nil && vp.validateOutput) {
				// if error or skip output validation, return output
				return
			} else if pout == nil {
				// validate output if available
				if len(out) > 0 && !miruken.IsNil(out[0]) {
					outcomeOut, poo, errOut := Validate(composer, out[0])
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
					if len(oo) > 0 && !miruken.IsNil(oo[0]) {
						outcomeOut, poo, errOut := Validate(composer, oo[0])
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
			// validate output if requested and available
			if vp.validateOutput && len(oo) > 0 && !miruken.IsNil(oo[0]) {
				outcomeOut, poo, errOut := Validate(composer, oo[0])
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

var _filters = []miruken.Filter{filter{}}