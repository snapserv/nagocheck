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
)

type swapPlugin struct {
	nagocheck.Plugin
}

type swapResource struct {
	nagocheck.Resource

	usagePercent float64
	usageStats   struct {
		totalBytes float64
		usedBytes  float64
		freeBytes  float64
	}
}

type swapSummarizer struct {
	nagocheck.Summarizer
}

func newSwapPlugin() *swapPlugin {
	return &swapPlugin{
		Plugin: nagocheck.NewPlugin("swap",
			nagocheck.PluginDescription("Swap Usage"),
		),
	}
}

func (p *swapPlugin) DefineCheck() nagopher.Check {
	check := nagopher.NewCheck("swap", newSwapSummarizer(p))
	check.AttachResources(newSwapResource(p))
	check.AttachContexts(
		nagopher.NewScalarContext(
			"usage",
			nagopher.OptionalBoundsPtr(p.WarningThreshold()),
			nagopher.OptionalBoundsPtr(p.CriticalThreshold()),
		),

		nagopher.NewScalarContext("total", nil, nil),
		nagopher.NewScalarContext("used", nil, nil),
		nagopher.NewScalarContext("free", nil, nil),
	)

	return check
}

func newSwapResource(plugin *swapPlugin) *swapResource {
	return &swapResource{
		Resource: nagocheck.NewResource(plugin),
	}
}

func (r *swapResource) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
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

	return metrics, nil
}

func (r *swapResource) Collect() error {
	swapStats, err := mem.SwapMemory()
	if err != nil {
		return err
	}

	r.usageStats.totalBytes = float64(swapStats.Total)
	r.usageStats.usedBytes = float64(swapStats.Used)
	r.usageStats.freeBytes = float64(swapStats.Free)

	r.usagePercent = nagocheck.Round(100-(r.usageStats.freeBytes/r.usageStats.totalBytes*100), 2)

	return nil
}

func newSwapSummarizer(plugin *swapPlugin) *swapSummarizer {
	return &swapSummarizer{
		Summarizer: nagocheck.NewSummarizer(plugin),
	}
}

func (s *swapSummarizer) Ok(check nagopher.Check) string {
	resultCollection := check.Results()

	return fmt.Sprintf(
		"%.2f%% used - Total:%s Used:%s",
		resultCollection.GetNumericMetricValue("usage").OrElse(math.NaN()),
		nagocheck.FormatBinarySize(resultCollection.GetNumericMetricValue("total").OrElse(math.NaN())),
		nagocheck.FormatBinarySize(resultCollection.GetNumericMetricValue("used").OrElse(math.NaN())),
	)
}
