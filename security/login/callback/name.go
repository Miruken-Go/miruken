package callback

// Name requests name information.
type Name struct {
	prompt      string
	defaultName string
	name        string
}

func (n *Name) Prompt() string {
	return n.prompt
}

func (n *Name) DefaultName() string {
	return n.defaultName
}

func (n *Name) Name() string {
	return n.name
}

func (n *Name) SetName(name string) {
	n.name = name
}

func NewName(prompt string, defaultName string) *Name {
	return &Name{prompt: prompt, defaultName: defaultName}
}
