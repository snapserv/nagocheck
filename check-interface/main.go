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
	"fmt"
	"math"
	"strconv"

	"github.com/snapserv/nagopher"
	"github.com/snapserv/nagopher-checks/shared"
	"gopkg.in/alecthomas/kingpin.v2"
)

type interfacePlugin struct {
	*shared.BasePlugin

	Name   string
	Speed  int
	Duplex string
}

type interfaceStats struct {
	State  string
	Speed  int
	Duplex string
}

type interfaceSummary struct {
	*nagopher.BaseSummary
}

func newInterfacePlugin() *interfacePlugin {
	return &interfacePlugin{
		BasePlugin: shared.NewPlugin(),
	}
}

func (p *interfacePlugin) ParseFlags() {
	kingpin.Flag("speed", "Return WARNING state when interface speed in Mbps does not match.").
		Short('s').
		IntVar(&p.Speed)

	kingpin.Flag("duplex", "Return WARNING state when interface duplex does not match (e.g.: half, full).").
		Short('d').
		HintOptions("half", "full").
		StringVar(&p.Duplex)

	kingpin.Arg("name", "Name of network interface.").
		Required().
		StringVar(&p.Name)

	p.BasePlugin.ParseFlags()
}

func (p *interfacePlugin) Probe(warnings *nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	interfaceStats, err := getInterfaceStats(p.Name)
	if err != nil {
		return metrics, err
	}

	metrics = append(metrics,
		nagopher.NewStringMetric("state", interfaceStats.State, ""),
		nagopher.NewStringMetric("duplex", interfaceStats.Duplex, ""),
		nagopher.NewNumericMetric("speed", float64(interfaceStats.Speed), "Mbps", nil, ""),
	)

	return metrics, nil
}

func newInterfaceSummary() *interfaceSummary {
	return &interfaceSummary{
		BaseSummary: nagopher.NewBaseSummary(),
	}
}

func (s *interfaceSummary) Ok(resultCollection *nagopher.ResultCollection) string {
	var interfaceSpeed string
	if value := s.GetNumericMetricValue(resultCollection, "speed", math.NaN()); math.IsNaN(value) {
		interfaceSpeed = "N/A"
	} else {
		interfaceSpeed = strconv.Itoa(int(value))
	}

	return fmt.Sprintf(
		"State:%s Speed:%s Duplex:%s",
		s.GetStringMetricValue(resultCollection, "state", "N/A"),
		interfaceSpeed,
		s.GetStringMetricValue(resultCollection, "duplex", "N/A"),
	)
}

func main() {
	plugin := newInterfacePlugin()
	plugin.ParseFlags()

	check := nagopher.NewCheck("interface", newInterfaceSummary())
	check.AttachResources(shared.NewPluginResource(plugin))
	check.AttachContexts(
		nagopher.NewContext("state", ""),
		nagopher.NewContext("duplex", ""),
		nagopher.NewContext("speed", ""),
	)

	plugin.Execute(check)
}
