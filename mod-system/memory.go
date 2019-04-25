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

type memoryPlugin struct {
	shared.Plugin
	CountReclaimable bool
}

type memoryUsage struct {
	active   float64
	buffers  float64
	cached   float64
	free     float64
	inactive float64
	total    float64
}

type memorySummary struct {
	shared.PluginSummarizer
}

func newMemoryPlugin() *memoryPlugin {
	return &memoryPlugin{
		Plugin: shared.NewPlugin(),
	}
}

func (p *memoryPlugin) DefineFlags(kp shared.KingpinNode) {
	p.Plugin.DefineDefaultFlags(kp)
	p.Plugin.DefineDefaultThresholds(kp)

	kp.Flag("count-reclaimable", "Count reclaimable space (cached/buffers) as used.").
		BoolVar(&p.CountReclaimable)
}

func (p *memoryPlugin) Execute() {
	check := nagopher.NewCheck("memory", newMemorySummary(p))
	check.AttachResources(shared.NewPluginResource(p))
	check.AttachContexts(
		nagopher.NewScalarContext(
			"usage",
			nagopher.OptionalBoundsPtr(p.WarningThreshold()),
			nagopher.OptionalBoundsPtr(p.CriticalThreshold()),
		),

		nagopher.NewScalarContext("active", nil, nil),
		nagopher.NewScalarContext("inactive", nil, nil),
		nagopher.NewScalarContext("buffers", nil, nil),
		nagopher.NewScalarContext("cached", nil, nil),
		nagopher.NewScalarContext("total", nil, nil),
	)

	p.ExecuteCheck(check)
}

func (p *memoryPlugin) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	valueRange, err := nagopher.NewBoundsFromNagiosRange("0:")
	if err != nil {
		return metrics, err
	}

	memoryUsage, err := getMemoryUsage()
	if err != nil {
		return metrics, err
	}

	freeMemory := memoryUsage.free
	if !p.CountReclaimable {
		freeMemory += memoryUsage.cached + memoryUsage.buffers
	}
	usagePercent := shared.Round(100-(freeMemory/memoryUsage.total*100), 2)

	metrics = append(metrics,
		nagopher.MustNewNumericMetric("usage", usagePercent, "%", nil, ""),

		nagopher.MustNewNumericMetric("active", memoryUsage.active, "B", &valueRange, ""),
		nagopher.MustNewNumericMetric("inactive", memoryUsage.inactive, "B", &valueRange, ""),
		nagopher.MustNewNumericMetric("buffers", memoryUsage.buffers, "B", &valueRange, ""),
		nagopher.MustNewNumericMetric("cached", memoryUsage.cached, "B", &valueRange, ""),
		nagopher.MustNewNumericMetric("total", memoryUsage.total, "B", &valueRange, ""),
	)

	return metrics, nil
}

func newMemorySummary(plugin *memoryPlugin) *memorySummary {
	return &memorySummary{
		PluginSummarizer: shared.NewPluginSummarizer(plugin),
	}
}

func (s *memorySummary) Ok(check nagopher.Check) string {
	resultCollection := check.Results()

	return fmt.Sprintf(
		"%.2f%% used - Total:%s Active:%s Inactive:%s Buffers:%s Cached:%s",

		resultCollection.GetNumericMetricValue("usage").OrElse(math.NaN()),
		shared.FormatBinarySize(resultCollection.GetNumericMetricValue("total").OrElse(math.NaN())),
		shared.FormatBinarySize(resultCollection.GetNumericMetricValue("active").OrElse(math.NaN())),
		shared.FormatBinarySize(resultCollection.GetNumericMetricValue("inactive").OrElse(math.NaN())),
		shared.FormatBinarySize(resultCollection.GetNumericMetricValue("buffers").OrElse(math.NaN())),
		shared.FormatBinarySize(resultCollection.GetNumericMetricValue("cached").OrElse(math.NaN())),
	)
}
