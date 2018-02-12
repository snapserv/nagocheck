/*
 * nagopher-checks - Reliable and lightweight Nagios plugins written in Go
 * Copyright (C) 2018  Pascal Mathis
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

package shared

import (
	"flag"

	"github.com/snapserv/nagopher"
)

// Plugin represents a interface for all plugin types.
type Plugin interface {
	ParseFlags()
	Probe(*nagopher.WarningCollection) ([]nagopher.Metric, error)
}

// BasePlugin represents a generic plugin from which all other plugin types should originate.
type BasePlugin struct {
	Verbose       bool
	WarningRange  *nagopher.Range
	CriticalRange *nagopher.Range
}

// BasePluginResource presents a generic nagopher 'BaseResource' linked to a plugin.
type BasePluginResource struct {
	*nagopher.BaseResource
	basePlugin Plugin
}

// NewPlugin instantiates 'BasePlugin'.
func NewPlugin() *BasePlugin {
	return &BasePlugin{}
}

// ParseFlags adds several flag definitions for parsing followed by executing 'flag.Parse()'. Afterwards, the threshold
// ranges (if given) are being parsed.
func (p *BasePlugin) ParseFlags() {
	var err error
	var warningRangeString, criticalRangeString string

	flag.BoolVar(&p.Verbose, "verbose", false, "Toggles verbose plugin output")
	flag.StringVar(&warningRangeString, "warning", "", "Warning threshold range specifier")
	flag.StringVar(&criticalRangeString, "critical", "", "Critical threshold range specifier")
	flag.Parse()

	if p.WarningRange, err = nagopher.ParseRange(warningRangeString); err != nil {
		panic(err.Error())
	}
	if p.CriticalRange, err = nagopher.ParseRange(criticalRangeString); err != nil {
		panic(err.Error())
	}
}

// Execute is a helper method which creates a new nagopher 'Runtime', executes a check and exits
func (p *BasePlugin) Execute(check *nagopher.Check) {
	runtime := nagopher.NewRuntime(p.Verbose)
	runtime.ExecuteAndExit(check)
}

// Probe represents the method executing the actual check/metrics logic and should be overridden by each plugin for
// returning metrics. It also supports adding warnings through the passed 'WarningCollection' or returning an error in
// case metric collection goes wrong.
func (p *BasePlugin) Probe(warnings *nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	return metrics, nil
}

// NewPluginResource instantiates 'BasePluginResource' and links it with the given plugin.
func NewPluginResource(plugin Plugin) *BasePluginResource {
	return &BasePluginResource{
		BaseResource: nagopher.NewResource(),
		basePlugin:   plugin,
	}
}

// Probe is an override for 'BaseResource.Probe(...)', which is being called by nagopher for collecting metrics. This
// method should never be overridden by any of the plugins, as it will just pass all arguments to 'Plugin.Probe()',
// where the plugins should define their actual check/metrics logic.
func (pr *BasePluginResource) Probe(warnings *nagopher.WarningCollection) ([]nagopher.Metric, error) {
	return pr.basePlugin.Probe(warnings)
}
