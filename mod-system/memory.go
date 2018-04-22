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

package modsystem

import (
	"fmt"
	"github.com/snapserv/nagopher"
	"github.com/snapserv/nagopher-checks/shared"
	"math"
	"strconv"
)

type memoryPlugin struct {
	*shared.BasePlugin

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
	*nagopher.BaseSummary
}

func newMemoryPlugin() *memoryPlugin {
	return &memoryPlugin{
		BasePlugin: shared.NewPlugin(),
	}
}

func (p *memoryPlugin) DefineFlags(kp shared.KingpinInterface) {
	p.BasePlugin.DefineFlags(kp, true)

	kp.Flag("count-reclaimable", "Count reclaimable space (cached/buffers) as used.").
		BoolVar(&p.CountReclaimable)
}

func (p *memoryPlugin) Execute() {
	check := nagopher.NewCheck("memory", newMemorySummary())
	check.AttachResources(shared.NewPluginResource(p))
	check.AttachContexts(
		nagopher.NewScalarContext(
			"usage",
			p.WarningRange,
			p.CriticalRange,
		),

		nagopher.NewScalarContext("active", nil, nil),
		nagopher.NewScalarContext("inactive", nil, nil),
		nagopher.NewScalarContext("buffers", nil, nil),
		nagopher.NewScalarContext("cached", nil, nil),
		nagopher.NewScalarContext("total", nil, nil),
	)

	p.ExecuteCheck(check)
}

func (p *memoryPlugin) Probe(warnings *nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	valueRange, err := nagopher.ParseRange("0:")
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
		nagopher.NewNumericMetric("usage", usagePercent, "%", nil, ""),

		nagopher.NewNumericMetric("active", memoryUsage.active, "KB", valueRange, ""),
		nagopher.NewNumericMetric("inactive", memoryUsage.inactive, "KB", valueRange, ""),
		nagopher.NewNumericMetric("buffers", memoryUsage.buffers, "KB", valueRange, ""),
		nagopher.NewNumericMetric("cached", memoryUsage.cached, "KB", valueRange, ""),
		nagopher.NewNumericMetric("total", memoryUsage.total, "KB", valueRange, ""),
	)

	return metrics, nil
}

func newMemorySummary() *memorySummary {
	return &memorySummary{
		BaseSummary: nagopher.NewBaseSummary(),
	}
}

func (s *memorySummary) getResultMetricValue(resultCollection *nagopher.ResultCollection, name string) float64 {
	result := resultCollection.GetByMetricName(name)
	if result != nil {
		if metric := result.Metric(); metric != nil {
			numberMetric := metric.(*nagopher.NumericMetric)
			return numberMetric.Value()
		}
	}

	return math.NaN()
}

func (s *memorySummary) Ok(resultCollection *nagopher.ResultCollection) string {
	return fmt.Sprintf(
		"%s%% used - Total:%s Active:%s Inactive:%s Buffers:%s Cached:%s",

		strconv.FormatFloat(
			s.getResultMetricValue(resultCollection, "usage"),
			'f', 2, strconv.IntSize,
		),

		shared.FormatBinarySize(s.getResultMetricValue(resultCollection, "total")),
		shared.FormatBinarySize(s.getResultMetricValue(resultCollection, "active")),
		shared.FormatBinarySize(s.getResultMetricValue(resultCollection, "inactive")),
		shared.FormatBinarySize(s.getResultMetricValue(resultCollection, "buffers")),
		shared.FormatBinarySize(s.getResultMetricValue(resultCollection, "cached")),
	)
}
