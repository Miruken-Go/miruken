package json

type (
	TypeFieldHandling uint

	TypeFieldInfo struct {
		Name  string
		Value string
	}
)

const (
	TypeFieldHandlingNone TypeFieldHandling = 0
	TypeFieldHandlingRoot TypeFieldHandling = 1 << iota
	TypeFieldHandlingInterfaces
)