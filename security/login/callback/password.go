package callback

import (
	"github.com/miruken-go/miruken"
)

type (
	// Password requests password information.
	Password struct {
		prompt   string
		display  bool
		password []byte
	}

	// PasswordHandler responds to Password callbacks with give password.
	PasswordHandler struct {
		Password []byte
	}
)


func (p *Password) Prompt() string {
	return p.prompt
}

func (p *Password) Display() bool {
	return p.display
}

func (p *Password) Password() []byte {
	return p.password
}

func (p *Password) SetPassword(password []byte) {
	p.password = password
}

func (p *Password) ClearPassword() {
	for i := range p.password {
		p.password[i] = 0
	}
}


// PasswordHandler

func (h PasswordHandler) Handle(
	c        any,
	greedy   bool,
	composer miruken.Handler,
) miruken.HandleResult {
	if n, ok := c.(*Password); ok {
		n.SetPassword(h.Password)
		return miruken.Handled
	}
	return miruken.NotHandled
}


func NewPassword(prompt string, display bool) *Password {
	return &Password{prompt: prompt, display: display}
}
