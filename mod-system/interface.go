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

package modsystem

import (
	"fmt"
	"github.com/snapserv/nagocheck/shared"
	"github.com/snapserv/nagopher"
	"math"
)

type interfacePlugin struct {
	shared.Plugin

	Name           string
	SpeedRange     nagopher.OptionalBounds
	ExpectedDuplex []string
}

type interfaceSummary struct {
	shared.PluginSummarizer
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
		Plugin: shared.NewPlugin(),
	}
}

func (p *interfacePlugin) DefineFlags(kp shared.KingpinNode) {
	p.Plugin.DefineDefaultFlags(kp)

	shared.NagopherBoundsVar(kp.Flag("speed",
		"Interface speed threshold formatted as Nagios range specifier.").Short('s'), &p.SpeedRange)

	kp.Flag("duplex", "Return WARNING state when interface duplex does not match (e.g.: half, full).").
		Short('d').
		HintOptions("half", "full").
		StringsVar(&p.ExpectedDuplex)

	kp.Arg("name", "Name of network interface.").
		Required().
		StringVar(&p.Name)
}

func (p *interfacePlugin) Execute() {
	store := &interfaceStore{}
	deltaRange, err := nagopher.NewBoundsFromNagiosRange("~:0")
	if err != nil {
		panic(err)
	}

	check := nagopher.NewCheck("interface", newInterfaceSummary(p))
	check.AttachResources(shared.NewPluginResource(p))
	check.AttachContexts(
		nagopher.NewStringMatchContext("state", nagopher.StateCritical(), []string{"UP"}),
		nagopher.NewStringMatchContext("duplex", nagopher.StateWarning(), p.ExpectedDuplex),
		nagopher.NewScalarContext("speed", nagopher.OptionalBoundsPtr(p.SpeedRange), nil),
		nagopher.NewDeltaContext("errors_tx", &store.PreviousTxErrors, &deltaRange, nil),
		nagopher.NewDeltaContext("errors_rx", &store.PreviousRxErrors, &deltaRange, nil),
	)

	p.ExecutePersistentCheck(check, "interface-"+p.Name, &store)
}

func (p *interfacePlugin) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
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
		nagopher.MustNewStringMetric("state", interfaceStats.State, ""),
		nagopher.MustNewStringMetric("duplex", interfaceStats.Duplex, ""),
		nagopher.MustNewNumericMetric("speed", intToFloat64(interfaceStats.Speed), "M", nil, ""),
		nagopher.MustNewNumericMetric("errors_tx", intToFloat64(interfaceStats.TxErrors), "c", nil, ""),
		nagopher.MustNewNumericMetric("errors_rx", intToFloat64(interfaceStats.RxErrors), "c", nil, ""),
	)

	return metrics, nil
}

func newInterfaceSummary(plugin *interfacePlugin) *interfaceSummary {
	return &interfaceSummary{
		PluginSummarizer: shared.NewPluginSummarizer(plugin),
	}
}

func (s *interfaceSummary) Ok(check nagopher.Check) string {
	var interfaceSpeed string = "N/A"
	resultCollection := check.Results()

	interfaceState := resultCollection.GetStringMetricValue("state").OrElse("N/A")
	interfaceDuplex := resultCollection.GetStringMetricValue("duplex").OrElse("N/A")

	speedMetric, err := resultCollection.GetMetricByName("speed").Get()
	if err == nil && speedMetric != nil {
		interfaceSpeed = speedMetric.ValueString() + speedMetric.ValueUnit()
	}

	return fmt.Sprintf("State:%s Speed:%s Duplex:%s", interfaceState, interfaceSpeed, interfaceDuplex)
}
