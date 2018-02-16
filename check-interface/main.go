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

	Name           string
	SpeedRange     *nagopher.Range
	ExpectedDuplex []string
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

func (p *interfacePlugin) ParseFlags() {
	var err error
	var speedRangeString string

	kingpin.Flag("speed", "Interface speed threshold formatted as Nagios range specifier.").
		Short('s').
		StringVar(&speedRangeString)

	kingpin.Flag("duplex", "Return WARNING state when interface duplex does not match (e.g.: half, full).").
		Short('d').
		HintOptions("half", "full").
		StringsVar(&p.ExpectedDuplex)

	kingpin.Arg("name", "Name of network interface.").
		Required().
		StringVar(&p.Name)

	p.BasePlugin.ParseFlags(false)

	if p.SpeedRange, err = nagopher.ParseRange(speedRangeString); err != nil {
		panic(err.Error())
	}
}

func (p *interfacePlugin) Probe(warnings *nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	interfaceStats, err := getInterfaceStats(p.Name, warnings)
	if err != nil {
		return metrics, err
	}

	intToFloat64 := func(value int) float64 {
		if value == -1 {
			return math.NaN()
		}

		return float64(value)
	}

	metrics = append(metrics,
		nagopher.NewStringMetric("state", interfaceStats.State, ""),
		nagopher.NewStringMetric("duplex", interfaceStats.Duplex, ""),
		nagopher.NewNumericMetric("speed", intToFloat64(interfaceStats.Speed), "M", nil, ""),
		nagopher.NewNumericMetric("errors_tx", intToFloat64(interfaceStats.TxErrors), "c", nil, ""),
		nagopher.NewNumericMetric("errors_rx", intToFloat64(interfaceStats.RxErrors), "c", nil, ""),
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
		if interfaceSpeed == "U" {
			interfaceSpeed = "N/A"
		}
	}

	interfaceDuplex := s.GetStringMetricValue(resultCollection, "duplex", "N/A")
	if interfaceDuplex == "" {
		interfaceDuplex = "N/A"
	}

	return fmt.Sprintf(
		"State:%s Speed:%s Duplex:%s",
		s.GetStringMetricValue(resultCollection, "state", "N/A"),
		interfaceSpeed, interfaceDuplex,
	)
}

func main() {
	store := &interfaceStore{}
	plugin := newInterfacePlugin()
	plugin.ParseFlags()

	deltaRange, err := nagopher.ParseRange("~:0")
	if err != nil {
		panic(err)
	}

	check := nagopher.NewCheck("interface", newInterfaceSummary())
	check.AttachResources(shared.NewPluginResource(plugin))
	check.AttachContexts(
		nagopher.NewStringMatchContext("state", []string{"UP"}, nagopher.StateCritical),
		nagopher.NewStringMatchContext("duplex", plugin.ExpectedDuplex, nagopher.StateWarning),
		nagopher.NewScalarContext("speed", plugin.SpeedRange, nil),
		nagopher.NewDeltaContext("errors_tx", &store.PreviousTxErrors, deltaRange, nil),
		nagopher.NewDeltaContext("errors_rx", &store.PreviousRxErrors, deltaRange, nil),
	)

	plugin.ExecutePersistent(check, "interface-"+plugin.Name, &store)
}
