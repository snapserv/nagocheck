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

type Plugin interface {
	Probe(*nagopher.WarningCollection) (error, []nagopher.Metric)
}

type BasePlugin struct {
	Verbose       bool
	WarningRange  *nagopher.Range
	CriticalRange *nagopher.Range
}

type BasePluginResource struct {
	*nagopher.BaseResource
	basePlugin Plugin
}

func NewPlugin() *BasePlugin {
	return &BasePlugin{}
}

func (p *BasePlugin) ParseFlags() {
	var err error
	var warningRangeString, criticalRangeString string

	flag.BoolVar(&p.Verbose, "verbose", false, "Toggles verbose plugin output")
	flag.StringVar(&warningRangeString, "warning", "", "Warning threshold range specifier")
	flag.StringVar(&criticalRangeString, "critical", "", "Critical threshold range specifier")
	flag.Parse()

	if err, p.WarningRange = nagopher.ParseRange(warningRangeString); err != nil {
		panic(err.Error())
	}
	if err, p.CriticalRange = nagopher.ParseRange(criticalRangeString); err != nil {
		panic(err.Error())
	}
}

func (p *BasePlugin) Execute(check *nagopher.Check) {
	runtime := nagopher.NewRuntime(p.Verbose)
	runtime.ExecuteAndExit(check)
}

func (p *BasePlugin) Probe(warnings *nagopher.WarningCollection) (_ error, metrics []nagopher.Metric) {
	return nil, metrics
}

func NewPluginResource(plugin Plugin) *BasePluginResource {
	return &BasePluginResource{
		BaseResource: nagopher.NewResource(),
		basePlugin:   plugin,
	}
}

func (pr *BasePluginResource) Probe(warnings *nagopher.WarningCollection) (error, []nagopher.Metric) {
	return pr.basePlugin.Probe(warnings)
}
