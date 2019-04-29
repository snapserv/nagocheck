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
	"github.com/snapserv/nagocheck/nagocheck"
	"github.com/snapserv/nagopher"
	"math"
)

type interfacePlugin struct {
	nagocheck.Plugin

	InterfaceName  string
	SpeedRange     nagopher.OptionalBounds
	ExpectedDuplex []string
}

type interfaceResource struct {
	nagocheck.Resource `json:"-"`

	linkState      string
	linkSpeed      int
	linkDuplex     string
	transmitErrors int
	receiveErrors  int

	PreviousTransmitErrors float64 `json:"txErrors"`
	PreviousReceiveErrors  float64 `json:"rxErrors"`
}

type interfaceSummarizer struct {
	nagocheck.Summarizer
}

func newInterfacePlugin() *interfacePlugin {
	return &interfacePlugin{
		Plugin: nagocheck.NewPlugin("interface",
			nagocheck.PluginDescription("Network Interface"),
			nagocheck.PluginDefaultThresholds(false),
		),
	}
}

func (p *interfacePlugin) DefineFlags(kp nagocheck.KingpinNode) {
	nagocheck.NagopherBoundsVar(kp.Flag("speed", "Interface speed threshold formatted as Nagios range specifier.").
		Short('s'), &p.SpeedRange)

	kp.Flag("duplex", "Return WARNING state when interface duplex does not match (e.g.: half, full).").
		Short('d').HintOptions("half", "full").StringsVar(&p.ExpectedDuplex)

	kp.Arg("name", "Name of network interface.").
		Required().StringVar(&p.InterfaceName)
}

func (p *interfacePlugin) DefineCheck() nagopher.Check {
	deltaRange := nagopher.NewBounds(nagopher.LowerBound(math.Inf(-1)), nagopher.UpperBound(0))
	resource := newInterfaceResource(p)

	check := nagopher.NewCheck("interface", newInterfaceSummarizer(p))
	check.AttachResources(resource)
	check.AttachContexts(
		nagopher.NewStringMatchContext("state", nagopher.StateCritical(), []string{"UP"}),
		nagopher.NewStringMatchContext("duplex", nagopher.StateWarning(), p.ExpectedDuplex),
		nagopher.NewScalarContext("speed", nagopher.OptionalBoundsPtr(p.SpeedRange), nil),
		nagopher.NewDeltaContext("errors_tx", &resource.PreviousReceiveErrors, &deltaRange, nil),
		nagopher.NewDeltaContext("errors_rx", &resource.PreviousTransmitErrors, &deltaRange, nil),
	)

	return check
}

func newInterfaceResource(plugin *interfacePlugin) *interfaceResource {
	resource := &interfaceResource{}
	resource.Resource = nagocheck.NewResource(plugin,
		nagocheck.ResourcePersistence(plugin.InterfaceName, &resource),
	)

	return resource
}

func (r *interfaceResource) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	if err := r.Collect(warnings); err != nil {
		return metrics, err
	}

	intToFloat64 := func(value int) float64 {
		if value == -1 {
			return math.NaN()
		}

		return float64(value)
	}

	metrics = append(metrics,
		nagopher.MustNewStringMetric("state", r.linkState, ""),
		nagopher.MustNewStringMetric("duplex", r.linkDuplex, ""),
		nagopher.MustNewNumericMetric("speed", intToFloat64(r.linkSpeed), "M", nil, ""),
		nagopher.MustNewNumericMetric("errors_tx", intToFloat64(r.transmitErrors), "c", nil, ""),
		nagopher.MustNewNumericMetric("errors_rx", intToFloat64(r.receiveErrors), "c", nil, ""),
	)

	return metrics, nil
}

func (r *interfaceResource) ThisPlugin() *interfacePlugin {
	return r.Resource.Plugin().(*interfacePlugin)
}

func newInterfaceSummarizer(plugin *interfacePlugin) *interfaceSummarizer {
	return &interfaceSummarizer{
		Summarizer: nagocheck.NewSummarizer(plugin),
	}
}

func (s *interfaceSummarizer) Ok(check nagopher.Check) string {
	var interfaceSpeed = "N/A"
	resultCollection := check.Results()

	interfaceState := resultCollection.GetStringMetricValue("state").OrElse("N/A")
	interfaceDuplex := resultCollection.GetStringMetricValue("duplex").OrElse("N/A")

	speedMetric, err := resultCollection.GetMetricByName("speed").Get()
	if err == nil && speedMetric != nil {
		interfaceSpeed = speedMetric.ValueString() + speedMetric.ValueUnit()
	}

	return fmt.Sprintf("State:%s Speed:%s Duplex:%s", interfaceState, interfaceSpeed, interfaceDuplex)
}
