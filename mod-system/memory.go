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

type memoryPlugin struct {
	nagocheck.Plugin

	CountReclaimable bool
}

type memoryResource struct {
	nagocheck.Resource

	usagePercentage float64
	usageStats      struct {
		active   float64
		buffers  float64
		cached   float64
		free     float64
		inactive float64
		total    float64
	}
}

type memorySummarizer struct {
	nagocheck.Summarizer
}

func newMemoryPlugin() *memoryPlugin {
	return &memoryPlugin{
		Plugin: nagocheck.NewPlugin("memory",
			nagocheck.PluginDescription("Memory Usage"),
		),
	}
}

func (p *memoryPlugin) DefineFlags(kp nagocheck.KingpinNode) {
	kp.Flag("count-reclaimable", "Count reclaimable space (cached/buffers) as used.").
		BoolVar(&p.CountReclaimable)
}

func (p *memoryPlugin) DefineCheck() nagopher.Check {
	check := nagopher.NewCheck("memory", newMemorySummarizer(p))
	check.AttachResources(newMemoryResource(p))
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

	return check
}

func newMemoryResource(plugin *memoryPlugin) *memoryResource {
	return &memoryResource{
		Resource: nagocheck.NewResource(plugin),
	}
}

func (r *memoryResource) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	valueRange, err := nagopher.NewBoundsFromNagiosRange("0:")
	if err != nil {
		return metrics, err
	}

	if err := r.Collect(); err != nil {
		return metrics, err
	}

	metrics = append(metrics,
		nagopher.MustNewNumericMetric("usage", r.usagePercentage, "%", nil, ""),

		nagopher.MustNewNumericMetric("active", r.usageStats.active, "B", &valueRange, ""),
		nagopher.MustNewNumericMetric("inactive", r.usageStats.inactive, "B", &valueRange, ""),
		nagopher.MustNewNumericMetric("buffers", r.usageStats.buffers, "B", &valueRange, ""),
		nagopher.MustNewNumericMetric("cached", r.usageStats.cached, "B", &valueRange, ""),
		nagopher.MustNewNumericMetric("total", r.usageStats.total, "B", &valueRange, ""),
	)

	return metrics, nil
}

func (r *memoryResource) ThisPlugin() *memoryPlugin {
	return r.Resource.Plugin().(*memoryPlugin)
}

func newMemorySummarizer(plugin *memoryPlugin) *memorySummarizer {
	return &memorySummarizer{
		Summarizer: nagocheck.NewSummarizer(plugin),
	}
}

func (s *memorySummarizer) Ok(check nagopher.Check) string {
	resultCollection := check.Results()

	return fmt.Sprintf(
		"%.2f%% used - Total:%s Active:%s Inactive:%s Buffers:%s Cached:%s",

		resultCollection.GetNumericMetricValue("usage").OrElse(math.NaN()),
		nagocheck.FormatBinarySize(resultCollection.GetNumericMetricValue("total").OrElse(math.NaN())),
		nagocheck.FormatBinarySize(resultCollection.GetNumericMetricValue("active").OrElse(math.NaN())),
		nagocheck.FormatBinarySize(resultCollection.GetNumericMetricValue("inactive").OrElse(math.NaN())),
		nagocheck.FormatBinarySize(resultCollection.GetNumericMetricValue("buffers").OrElse(math.NaN())),
		nagocheck.FormatBinarySize(resultCollection.GetNumericMetricValue("cached").OrElse(math.NaN())),
	)
}
