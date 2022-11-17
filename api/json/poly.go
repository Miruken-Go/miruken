package json

type (
	TypeFieldHandling uint8

	TypeFieldInfo struct {
		Name  string
		Value string
	}
)

const (
	TypeFieldHandlingNone TypeFieldHandling = 0
	TypeFieldHandlingRoot TypeFieldHandling = 1 << iota
)