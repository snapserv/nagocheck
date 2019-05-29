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
	"github.com/shirou/gopsutil/mem"
	"github.com/snapserv/nagocheck/nagocheck"
	"github.com/snapserv/nagopher"
	"math"
	"strings"
)

type memoryPlugin struct {
	nagocheck.Plugin

	CountReclaimable bool
}

type memoryResource struct {
	nagocheck.Resource

	usagePercent float64
	usageStats   struct {
		totalBytes float64
		usedBytes  float64
		freeBytes  float64

		activeBytes   float64
		inactiveBytes float64
		wiredBytes    float64
		buffersBytes  float64
		cachedBytes   float64
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
	kp.Flag("count-reclaimable", "Count reclaimable space (cached/buffers) as usedBytes.").
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

		nagopher.NewScalarContext("total", nil, nil),
		nagopher.NewScalarContext("used", nil, nil),
		nagopher.NewScalarContext("free", nil, nil),

		nagopher.NewScalarContext("active", nil, nil),
		nagopher.NewScalarContext("inactive", nil, nil),
		nagopher.NewScalarContext("wired", nil, nil),
		nagopher.NewScalarContext("buffers", nil, nil),
		nagopher.NewScalarContext("cached", nil, nil),
	)

	return check
}

func newMemoryResource(plugin *memoryPlugin) *memoryResource {
	return &memoryResource{
		Resource: nagocheck.NewResource(plugin),
	}
}

func (r *memoryResource) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	valueRange := nagopher.NewBounds(nagopher.BoundsOpt(nagopher.LowerBound(0)))

	if err := r.Collect(); err != nil {
		return metrics, err
	}

	metrics = append(metrics,
		nagopher.MustNewNumericMetric("usage", r.usagePercent, "%", nil, ""),
		nagopher.MustNewNumericMetric("total", r.usageStats.totalBytes, "B", &valueRange, ""),
		nagopher.MustNewNumericMetric("used", r.usageStats.usedBytes, "B", &valueRange, ""),
		nagopher.MustNewNumericMetric("free", r.usageStats.freeBytes, "B", &valueRange, ""),
	)

	optionalMetric := func(name string, value float64, valueUnit string, valueRange *nagopher.Bounds, context string) {
		if !math.IsNaN(value) && value != 0 {
			metrics = append(metrics, nagopher.MustNewNumericMetric(name, value, valueUnit, valueRange, context))
		}
	}

	optionalMetric("active", r.usageStats.activeBytes, "B", &valueRange, "")
	optionalMetric("inactive", r.usageStats.inactiveBytes, "B", &valueRange, "")
	optionalMetric("wired", r.usageStats.wiredBytes, "B", &valueRange, "")
	optionalMetric("buffers", r.usageStats.buffersBytes, "B", &valueRange, "")
	optionalMetric("cached", r.usageStats.cachedBytes, "B", &valueRange, "")

	return metrics, nil
}

func (r *memoryResource) Collect() error {
	vmStats, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	freeBytes := vmStats.Free
	if !r.ThisPlugin().CountReclaimable {
		freeBytes += vmStats.Cached + vmStats.Buffers
	}

	r.usageStats.totalBytes = float64(vmStats.Total)
	r.usageStats.usedBytes = float64(vmStats.Total - freeBytes)
	r.usageStats.freeBytes = float64(freeBytes)

	r.usageStats.activeBytes = float64(vmStats.Active)
	r.usageStats.inactiveBytes = float64(vmStats.Inactive)
	r.usageStats.wiredBytes = float64(vmStats.Wired)
	r.usageStats.buffersBytes = float64(vmStats.Buffers)
	r.usageStats.cachedBytes = float64(vmStats.Cached)

	r.usagePercent = nagocheck.Round(100-(r.usageStats.freeBytes/r.usageStats.totalBytes*100), 2)

	return nil
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
	result := fmt.Sprintf(
		"%.2f%% used - Total:%s Used:%s",
		resultCollection.GetNumericMetricValue("usage").OrElse(math.NaN()),
		nagocheck.FormatBinarySize(resultCollection.GetNumericMetricValue("total").OrElse(math.NaN())),
		nagocheck.FormatBinarySize(resultCollection.GetNumericMetricValue("used").OrElse(math.NaN())),
	)

	optionalResult := func(metricName string) {
		numericMetric, err := resultCollection.GetNumericMetricValue(metricName).Get()
		if err != nil {
			return
		}

		result += fmt.Sprintf(" %s:%s", strings.Title(metricName), nagocheck.FormatBinarySize(numericMetric))
	}

	optionalResult("buffers")
	optionalResult("cached")

	return result
}
