/*
 * nagocheck - Reliable and lightweight Nagios plugins written in Go
 * Copyright (C) 2018-2019  Pascal Mathis
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package nagocheck

import (
	"github.com/snapserv/nagopher"
)

// Plugin represents a single check including its CLI arguments
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

// PluginOpt is a type alias for functional options used by NewPlugin()
type PluginOpt func(*basePlugin)

type basePlugin struct {
	name                 string
	description          string
	module               Module
	useDefaultFlags      bool
	useDefaultThresholds bool
	forceVerboseOutput   bool

	verboseOutput     bool
	warningThreshold  nagopher.OptionalBounds
	criticalThreshold nagopher.OptionalBounds
}

// NewPlugin instantiates basePlugin with the given functional options
func NewPlugin(name string, options ...PluginOpt) Plugin {
	plugin := &basePlugin{
		name:                 name,
		description:          name,
		useDefaultFlags:      true,
		useDefaultThresholds: true,
		forceVerboseOutput:   false,
	}

	for _, option := range options {
		option(plugin)
	}

	return plugin
}

// PluginDescription is a functional option for NewPlugin(), which sets the module description
func PluginDescription(description string) PluginOpt {
	return func(p *basePlugin) {
		p.description = description
	}
}

// PluginForceVerbose is a functional option for NewPlugin(), which toggles forcing verbose check output
func PluginForceVerbose(enabled bool) PluginOpt {
	return func(p *basePlugin) {
		p.forceVerboseOutput = enabled
	}
}

// PluginDefaultFlags is a functional option for NewPlugin(), which toggles the definition of default flags
func PluginDefaultFlags(enabled bool) PluginOpt {
	return func(p *basePlugin) {
		p.useDefaultFlags = enabled
	}
}

// PluginDefaultThresholds is a functional option for NewPlugin(), which toggles the definition of default thresholds
func PluginDefaultThresholds(enabled bool) PluginOpt {
	return func(p *basePlugin) {
		p.useDefaultThresholds = enabled
	}
}

func (p *basePlugin) defineDefaultFlags(node KingpinNode) {
	if p.useDefaultFlags {
		if !p.verboseOutput {
			node.Flag("verbose", "Enable verbose plugin output.").
				Short('v').BoolVar(&p.verboseOutput)
		}
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
	if p.forceVerboseOutput {
		return true
	}

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
