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

	"github.com/snapserv/nagopher"
	"github.com/snapserv/nagopher-checks/shared"
	"gopkg.in/alecthomas/kingpin.v2"
)

type interfacePlugin struct {
	*shared.BasePlugin

	Name             string
	SpeedRange       *nagopher.Range
	speedRangeString string
	ExpectedDuplex   []string
}

type interfaceSummary struct {
	*nagopher.BaseSummary
}

type interfaceStats struct {
	State    string
	Speed    int
	Duplex   string
	TxErrors int
	RxErrors int
}

type interfaceStore struct {
	PreviousTxErrors float64
	PreviousRxErrors float64
}

func newInterfacePlugin() *interfacePlugin {
	return &interfacePlugin{
		BasePlugin: shared.NewPlugin(),
	}
}

func (p *interfacePlugin) DefineFlags() {
	kingpin.Flag("speed", "Interface speed threshold formatted as Nagios range specifier.").
		Short('s').
		StringVar(&p.speedRangeString)

	kingpin.Flag("duplex", "Return WARNING state when interface duplex does not match (e.g.: half, full).").
		Short('d').
		HintOptions("half", "full").
		StringsVar(&p.ExpectedDuplex)

	kingpin.Arg("name", "Name of network interface.").
		Required().
		StringVar(&p.Name)
}

func (p *interfacePlugin) ParseFlags() {
	var err error

	p.BasePlugin.ParseFlags()

	if p.SpeedRange, err = nagopher.ParseRange(p.speedRangeString); err != nil {
		panic(err.Error())
	}
}

func (p *interfacePlugin) Probe(warnings *nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	interfaceStats, err := getInterfaceStats(p.Name)
	if err != nil {
		return metrics, err
	}

	interfaceSpeed := float64(interfaceStats.Speed)
	if interfaceStats.Speed == -1 {
		interfaceSpeed = math.NaN()
	}

	metrics = append(metrics,
		nagopher.NewStringMetric("state", interfaceStats.State, ""),
		nagopher.NewStringMetric("duplex", interfaceStats.Duplex, ""),
		nagopher.NewNumericMetric("speed", interfaceSpeed, "M", nil, ""),
		nagopher.NewNumericMetric("errors_tx", float64(interfaceStats.TxErrors), "c", nil, ""),
		nagopher.NewNumericMetric("errors_rx", float64(interfaceStats.RxErrors), "c", nil, ""),
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

	speedResult := resultCollection.GetByMetricName("speed")
	if speedResult != nil {
		interfaceSpeed = speedResult.Metric().ValueUnit()
	}

	return fmt.Sprintf(
		"State:%s Speed:%s Duplex:%s",
		s.GetStringMetricValue(resultCollection, "state", "N/A"),
		interfaceSpeed,
		s.GetStringMetricValue(resultCollection, "duplex", "N/A"),
	)
}

func main() {
	store := &interfaceStore{}

	plugin := newInterfacePlugin()
	plugin.DefineFlags()

	plugin.ParseFlags()

	check := nagopher.NewCheck("interface", newInterfaceSummary())
	check.AttachResources(shared.NewPluginResource(plugin))
	check.AttachContexts(
		nagopher.NewStringMatchContext("state", []string{"UP"}, nagopher.StateWarning),
		nagopher.NewStringMatchContext("duplex", plugin.ExpectedDuplex, nagopher.StateWarning),
		nagopher.NewScalarContext("speed", nil, plugin.SpeedRange),
		nagopher.NewDeltaContext("errors_tx", &store.PreviousTxErrors, nil, nil),
		nagopher.NewDeltaContext("errors_rx", &store.PreviousRxErrors, nil, nil),
	)

	plugin.ExecutePersistent(check, "interface-"+plugin.Name, &store)
}
