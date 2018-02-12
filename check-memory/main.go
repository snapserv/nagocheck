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

	"fmt"

	"math"

	"strconv"

	"github.com/snapserv/nagopher"
	"github.com/snapserv/nagopher-checks/shared"
)

type memoryPlugin struct {
	*shared.BasePlugin

	IgnoreReclaimable bool
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

func (p *memoryPlugin) ParseFlags() {
	flag.BoolVar(&p.IgnoreReclaimable, "ignore-reclaimable", false,
		"Toggles ignoring reclaimable space in usage calculation (counts cached/buffers as used)")

	p.BasePlugin.ParseFlags()
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
	if !p.IgnoreReclaimable {
		freeMemory += memoryUsage.cached + memoryUsage.buffers
	}
	usagePercent := shared.Round(100-(freeMemory/memoryUsage.total*100), 2)

	metrics = append(metrics,
		nagopher.NewMetric("usage", usagePercent, "%", nil, ""),

		nagopher.NewMetric("active", memoryUsage.active, "KB", valueRange, ""),
		nagopher.NewMetric("inactive", memoryUsage.inactive, "KB", valueRange, ""),
		nagopher.NewMetric("buffers", memoryUsage.buffers, "KB", valueRange, ""),
		nagopher.NewMetric("cached", memoryUsage.cached, "KB", valueRange, ""),
		nagopher.NewMetric("total", memoryUsage.total, "KB", valueRange, ""),
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
	if result != nil && result.Metric() != nil {
		return result.Metric().Value()
	}

	return math.NaN()
}

func (s *memorySummary) formatSize(size float64) string {
	units := []struct {
		Divisor float64
		Suffix  string
	}{
		{math.Pow(1024, 3), "T"},
		{math.Pow(1024, 2), "G"},
		{math.Pow(1024, 1), "M"},
		{0, "K"},
	}

	if !math.IsNaN(size) {
		for _, unit := range units {
			if size > unit.Divisor*100 {
				value := shared.Round(size/unit.Divisor, 2)
				return strconv.FormatFloat(value, 'f', -1, strconv.IntSize) + unit.Suffix
			}
		}
	}

	return "N/A"
}

func (s *memorySummary) Ok(resultCollection *nagopher.ResultCollection) string {
	return fmt.Sprintf(
		"%s%% used - Total:%s Active:%s Inactive:%s Buffers:%s Cached:%s",

		strconv.FormatFloat(
			s.getResultMetricValue(resultCollection, "usage"),
			'f', -1, strconv.IntSize,
		),

		s.formatSize(s.getResultMetricValue(resultCollection, "total")),
		s.formatSize(s.getResultMetricValue(resultCollection, "active")),
		s.formatSize(s.getResultMetricValue(resultCollection, "inactive")),
		s.formatSize(s.getResultMetricValue(resultCollection, "buffers")),
		s.formatSize(s.getResultMetricValue(resultCollection, "cached")),
	)
}

func main() {
	plugin := newMemoryPlugin()
	plugin.ParseFlags()

	check := nagopher.NewCheck("memory", newMemorySummary())
	check.AttachResources(shared.NewPluginResource(plugin))
	check.AttachContexts(
		nagopher.NewScalarContext(
			"usage",
			plugin.WarningRange,
			plugin.CriticalRange,
		),

		nagopher.NewScalarContext("active", nil, nil),
		nagopher.NewScalarContext("inactive", nil, nil),
		nagopher.NewScalarContext("buffers", nil, nil),
		nagopher.NewScalarContext("cached", nil, nil),
		nagopher.NewScalarContext("total", nil, nil),
	)

	plugin.Execute(check)
}
