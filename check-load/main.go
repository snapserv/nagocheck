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

package main

import (
	"flag"
	"github.com/snapserv/nagopher"
	"github.com/snapserv/nagopher-checks/shared"
	"runtime"
)

type LoadPlugin struct {
	*shared.BasePlugin

	PerCPU bool
}

func NewLoadPlugin() *LoadPlugin {
	return &LoadPlugin{
		BasePlugin: shared.NewPlugin(),
	}
}

func (p *LoadPlugin) ParseFlags() {
	flag.BoolVar(&p.PerCPU, "per-cpu", false,
		"Toggles per-cpu metrics (divides load average by cpu count)")

	p.BasePlugin.ParseFlags()
}

func (p *LoadPlugin) Probe(warnings *nagopher.WarningCollection) (_ error, metrics []nagopher.Metric) {
	err, valueRange := nagopher.ParseRange("0:")
	if err != nil {
		return err, metrics
	}

	err, loadAverages := getLoadAverages()
	if err != nil {
		return err, metrics
	}

	metricNames := []string{"load1", "load5", "load15"}
	cpuCount := runtime.NumCPU()

	for key, loadAverage := range loadAverages {
		if p.PerCPU {
			loadAverage /= float64(cpuCount)
		}

		metrics = append(metrics, nagopher.NewMetric(
			metricNames[key], loadAverage, "", valueRange, "load",
		))
	}

	return nil, metrics
}

func main() {
	plugin := NewLoadPlugin()
	plugin.ParseFlags()

	check := nagopher.NewCheck("load", nagopher.NewBaseSummary())
	check.AttachResources(shared.NewPluginResource(plugin))
	check.AttachContexts(
		nagopher.NewScalarContext(
			"load",
			plugin.WarningRange,
			plugin.CriticalRange,
		),
	)

	plugin.Execute(check)
}
