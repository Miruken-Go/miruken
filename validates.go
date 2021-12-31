package miruken

// Validates callbacks contravariantly.
type Validates struct {
	CallbackBase
	target interface{}
}

func (v *Validates) Target() interface{} {
	return v.target
}
