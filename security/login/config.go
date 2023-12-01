package login

import "errors"

type (
	// ModuleEntry defines a single module in the
	// login Configuration.
	ModuleEntry struct {
		Module  string
		Options map[string]any
	}

	// Flow lists the modules in an authentication process.
	Flow []ModuleEntry

	// Configuration defines available authentication flows.
	Configuration map[string]Flow
)


func (m ModuleEntry) Validate() error {
	if m.Module == "" {
		return errors.New("module is required")
	}
	return nil
}


func (f Flow) Validate() error {
	if len(f) == 0 {
		return errors.New("flow requires at least one module")
	}
	for _, entry := range f {
		if err := entry.Validate(); err != nil {
			return err
		}
	}
	return nil
}