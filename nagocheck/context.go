package nagocheck

import "github.com/snapserv/nagopher"

// Context provides a base type for nagocheck contexts, which embeds nagopher.Context
type Context interface {
	nagopher.Context
	Plugin() Plugin
}

// ContextOpt is a type alias for functional options used by NewContext()
type ContextOpt func(*baseContext)

type baseContext struct {
	nagopher.Context
	plugin Plugin
}

// NewContext instantiates baseContext with the given functional options
func NewContext(plugin Plugin, parentContext nagopher.Context, options ...ContextOpt) Context {
	context := &baseContext{
		Context: parentContext,
		plugin:  plugin,
	}

	for _, option := range options {
		option(context)
	}

	return context
}

func (s *baseContext) Plugin() Plugin {
	return s.plugin
}
