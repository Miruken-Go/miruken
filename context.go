package miruken

import "sync"

// ContextState represents the state of a Context.
type ContextState uint

const (
	ContextActive ContextState = iota
	ContextEnding
	ContextEnded
)

type Context struct {
	Handler
	state     ContextState
	children *[]Context
	lock      sync.RWMutex
}



