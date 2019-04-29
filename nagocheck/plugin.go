package nagocheck

import (
	"github.com/snapserv/nagopher"
)

type Plugin interface {
	Name() string
	Description() string
	Module() Module
	DefineFlags(node KingpinNode)
	DefineCheck() nagopher.Check

	VerboseOutput() bool
	WarningThreshold() nagopher.OptionalBounds
	CriticalThreshold() nagopher.OptionalBounds

	setModule(module Module)
	defineDefaultFlags(node KingpinNode)
}

type PluginOpt func(*basePlugin)

type basePlugin struct {
	name                 string
	description          string
	module               Module
	useDefaultFlags      bool
	useDefaultThresholds bool

	verboseOutput     bool
	warningThreshold  nagopher.OptionalBounds
	criticalThreshold nagopher.OptionalBounds
}

func NewPlugin(name string, options ...PluginOpt) Plugin {
	plugin := &basePlugin{
		name:                 name,
		description:          name,
		useDefaultFlags:      true,
		useDefaultThresholds: true,
	}

	for _, option := range options {
		option(plugin)
	}

	return plugin
}

func PluginDescription(description string) PluginOpt {
	return func(p *basePlugin) {
		p.description = description
	}
}

func PluginDefaultFlags(enabled bool) PluginOpt {
	return func(p *basePlugin) {
		p.useDefaultFlags = enabled
	}
}

func PluginDefaultThresholds(enabled bool) PluginOpt {
	return func(p *basePlugin) {
		p.useDefaultThresholds = enabled
	}
}

func (p *basePlugin) defineDefaultFlags(node KingpinNode) {
	if p.useDefaultFlags {
		node.Flag("verbose", "Enable verbose plugin output.").
			Short('v').BoolVar(&p.verboseOutput)
	}

	if p.useDefaultThresholds {
		NagopherBoundsVar(node.Flag("warning", "Warning threshold formatted as Nagios range specifier.").
			Short('w'), &p.warningThreshold)
		NagopherBoundsVar(node.Flag("critical", "Critical threshold formatted as Nagios range specifier.").
			Short('c'), &p.criticalThreshold)
	}
}

func (p *basePlugin) Name() string {
	return p.name
}

func (p *basePlugin) Description() string {
	return p.description
}

func (p *basePlugin) Module() Module {
	return p.module
}

func (p *basePlugin) setModule(module Module) {
	p.module = module
}

func (p *basePlugin) VerboseOutput() bool {
	return p.verboseOutput
}

func (p *basePlugin) WarningThreshold() nagopher.OptionalBounds {
	return p.warningThreshold
}

func (p *basePlugin) CriticalThreshold() nagopher.OptionalBounds {
	return p.criticalThreshold
}

func (p *basePlugin) DefineFlags(node KingpinNode) {}

func (p *basePlugin) DefineCheck() nagopher.Check {
	return nagopher.NewCheck(p.name, NewSummarizer(p))
}
