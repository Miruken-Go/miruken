package miruken

type Resolving struct {
	*Inquiry
	callback interface{}
}

func NewResolving(key interface{}, callback interface{}) *Resolving {
	parent, _ := callback.(*Inquiry)
	inquiry := NewInquiry(key, true, parent)
	return &Resolving{inquiry, callback}
}
