package login

type (
	// ModuleEntry defines a single module in the
	// login Configuration.
	ModuleEntry struct {
		Key     string
		Options map[string]any
	}

	// Flow lists the modules in an authentication process.
	Flow []ModuleEntry

	// Configuration defines available authentication flows.
	Configuration map[string]Flow
)
