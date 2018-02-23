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
	"github.com/snapserv/nagopher"
	"github.com/snapserv/nagopher-checks/shared"
	"gopkg.in/alecthomas/kingpin.v2"
	"runtime"
)

type loadPlugin struct {
	*shared.BasePlugin

	PerCPU bool
}

func newLoadPlugin() *loadPlugin {
	return &loadPlugin{
		BasePlugin: shared.NewPlugin(),
	}
}

func (p *loadPlugin) DefineFlags(kp shared.KingpinInterface) {
	kp.Flag("per-cpu", "Enable per-cpu metrics (divide load average by cpu count).").BoolVar(&p.PerCPU)
}

func (p *loadPlugin) Execute() {
	check := nagopher.NewCheck("load", nagopher.NewBaseSummary())
	check.AttachResources(shared.NewPluginResource(p))
	check.AttachContexts(
		nagopher.NewScalarContext(
			"load",
			p.WarningRange,
			p.CriticalRange,
		),
	)

	p.ExecuteCheck(check)
}

func (p *loadPlugin) Probe(warnings *nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	valueRange, err := nagopher.ParseRange("0:")
	if err != nil {
		return metrics, err
	}

	loadAverages, err := getLoadAverages()
	if err != nil {
		return metrics, err
	}

	metricNames := []string{"load1", "load5", "load15"}
	cpuCount := runtime.NumCPU()

	for key, loadAverage := range loadAverages {
		if p.PerCPU {
			loadAverage /= float64(cpuCount)
		}

		metrics = append(metrics, nagopher.NewNumericMetric(
			metricNames[key], loadAverage, "", valueRange, "load",
		))
	}

	return metrics, nil
}

func main() {
	plugin := newLoadPlugin()
	plugin.DefineFlags(kingpin.CommandLine)
	kingpin.Parse()
	plugin.Execute()
}
