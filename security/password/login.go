package password

import (
	"bytes"
	"errors"
	"strings"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/security"
	"github.com/miruken-go/miruken/security/login/callback"
	"github.com/miruken-go/miruken/security/principal"
)

type (
	// Verifier checks the password for a username.
	Verifier interface {
		VerifyPassword(username string, password []byte) bool
	}

	// Map is a Verifier backed by a map of username and passwords.
	Map map[string][]byte

	// LoginModule authenticates a subject with username and password.
	LoginModule struct {
		verifiers []Verifier
		user      principal.User
		password  *callback.Password
	}
)

var (
	ErrInvalidCredentialsOption = errors.New("invalid credentials option")
	ErrInvalidUsername          = errors.New("invalid username option")
	ErrInvalidPassword          = errors.New("invalid password option")
	ErrMissingUsername          = errors.New("missing username")
	ErrMissingPassword          = errors.New("missing password")
)

// Map

func (m Map) VerifyPassword(username string, password []byte) bool {
	if pwd, ok := m[username]; ok {
		return bytes.Equal(pwd, password)
	}
	return false
}

// LoginModule

func (l *LoginModule) Constructor(
	_ *struct {
		creates.It `key:"login.pwd"`
	},
	verifiers []Verifier,
) {
	l.verifiers = verifiers
}

func (l *LoginModule) Init(opts map[string]any) error {
	for k, opt := range opts {
		switch strings.ToLower(k) {
		case "credentials":
			var credentials Map
			switch o := opt.(type) {
			case map[string]any:
				credentials = make(Map, len(o))
				for u, p := range o {
					username := strings.TrimSpace(u)
					if username == "" {
						return ErrInvalidUsername
					}
					if pwd, ok := p.(string); !ok {
						return ErrInvalidPassword
					} else {
						pwd = strings.TrimSpace(pwd)
						if pwd == "" {
							return ErrInvalidPassword
						}
						credentials[username] = []byte(pwd)
					}
				}
			case []any:
				credentials = make(Map, len(o))
				for _, userPasswords := range o {
					if credential, ok := userPasswords.(map[string]any); !ok {
						return ErrInvalidCredentialsOption
					} else {
						var username, password string
						for pk, pv := range credential {
							switch strings.ToLower(pk) {
							case "username":
								if u, ok := pv.(string); !ok {
									return ErrInvalidUsername
								} else {
									username = u
								}
							case "password":
								if p, ok := pv.(string); !ok {
									return ErrInvalidPassword
								} else {
									password = p
								}
							}
						}
						username = strings.TrimSpace(username)
						if username == "" {
							return ErrInvalidUsername
						}
						password = strings.TrimSpace(password)
						if password == "" {
							return ErrInvalidPassword
						}
						credentials[username] = []byte(password)
					}
				}
			default:
				return ErrInvalidCredentialsOption
			}
			l.verifiers = append(l.verifiers, credentials)
		}
	}
	return nil
}

func (l *LoginModule) Login(
	subject security.Subject,
	handler miruken.Handler,
) error {
	username := callback.NewName("prompt", "")
	if !handler.Handle(username, false, nil).Handled() {
		return ErrMissingUsername
	}

	password := callback.NewPassword("prompt", false)
	if !handler.Handle(password, false, nil).Handled() {
		return ErrMissingPassword
	}

	user := username.Name()
	for _, verifier := range l.verifiers {
		if verifier.VerifyPassword(user, password.Password()) {
			l.user = principal.User(user)
			l.password = password
			subject.AddPrincipals(l.user)
			subject.AddCredentials(l.password)
			break
		}
	}

	return nil
}

func (l *LoginModule) Logout(
	subject security.Subject,
	handler miruken.Handler,
) error {
	subject.RemovePrincipals(l.user)
	subject.RemoveCredentials(l.password)
	l.password.ClearPassword()
	return nil
}
