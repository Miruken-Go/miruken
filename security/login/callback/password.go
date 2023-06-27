package callback

// Password requests password information.
type Password struct {
	prompt   string
	display  bool
	password []byte
}


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
	for i, _ := range p.password {
		p.password[i] = 0
	}
}


func NewPassword(prompt string, display bool) *Password {
	return &Password{prompt: prompt, display: display}
}
