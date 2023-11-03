package callback

import "github.com/miruken-go/miruken"

type (
	// Name requests name information.
	Name struct {
		prompt      string
		defaultName string
		name        string
	}

	// NameHandler responds to Name callbacks with give name.
	NameHandler struct {
		Name string
	}
)


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


// NameHandler

func (h NameHandler) Handle(
	c        any,
	greedy   bool,
	composer miruken.Handler,
) miruken.HandleResult {
	if n, ok := c.(*Name); ok {
		n.SetName(h.Name)
		return miruken.Handled
	}
	return miruken.NotHandled
}


func NewName(prompt string, defaultName string) *Name {
	return &Name{prompt: prompt, defaultName: defaultName}
}
