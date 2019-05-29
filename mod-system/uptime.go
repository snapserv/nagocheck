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
	"github.com/shirou/gopsutil/host"
	"github.com/snapserv/nagocheck/nagocheck"
	"github.com/snapserv/nagopher"
	"math"
	"strconv"
	"time"
)

type uptimePlugin struct {
	nagocheck.Plugin
}

type uptimeResource struct {
	nagocheck.Resource

	uptime float64
}

type uptimeContext struct {
	nagocheck.Context

	warningThreshold  nagopher.OptionalBounds
	criticalThreshold nagopher.OptionalBounds
}

type uptimeSummarizer struct {
	nagocheck.Summarizer
}

func newUptimePlugin() *uptimePlugin {
	return &uptimePlugin{
		Plugin: nagocheck.NewPlugin("uptime",
			nagocheck.PluginDescription("System Uptime"),
		),
	}
}

func (p *uptimePlugin) DefineCheck() nagopher.Check {
	check := nagopher.NewCheck("uptime", newUptimeSummarizer(p))
	check.AttachResources(newUptimeResource(p))
	check.AttachContexts(
		newUptimeContext(p,
			nagopher.OptionalBoundsPtr(p.WarningThreshold()),
			nagopher.OptionalBoundsPtr(p.CriticalThreshold()),
		),
	)

	return check
}

func newUptimeResource(plugin *uptimePlugin) *uptimeResource {
	return &uptimeResource{
		Resource: nagocheck.NewResource(plugin),
	}
}

func (r *uptimeResource) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	valueRange := nagopher.NewBounds(nagopher.BoundsOpt(nagopher.LowerBound(0)))

	if err := r.Collect(); err != nil {
		return metrics, err
	}

	metrics = append(metrics,
		nagopher.MustNewNumericMetric("uptime", r.uptime, "s", &valueRange, ""),
	)

	return metrics, nil
}

func (r *uptimeResource) Collect() error {
	uptime, err := host.Uptime()
	if err != nil {
		return err
	}

	r.uptime = float64(uptime)

	return nil
}

func newUptimeContext(plugin *uptimePlugin, warningThreshold *nagopher.Bounds, criticalThreshold *nagopher.Bounds) *uptimeContext {
	uptimeContext := &uptimeContext{
		Context: nagocheck.NewContext(plugin, nagopher.NewBaseContext("uptime", "%<value>s")),
	}

	if warningThreshold != nil {
		uptimeContext.warningThreshold = nagopher.NewOptionalBounds(*warningThreshold)
	}
	if criticalThreshold != nil {
		uptimeContext.criticalThreshold = nagopher.NewOptionalBounds(*criticalThreshold)
	}

	return uptimeContext
}

func (c *uptimeContext) Describe(metric nagopher.Metric) string {
	uptimeMetric, ok := metric.(nagopher.NumericMetric)
	if !ok {
		return c.Context.Describe(metric)
	}

	uptimeSeconds := int64(uptimeMetric.Value())
	uptimeDuration, err := time.ParseDuration(strconv.FormatInt(uptimeSeconds, 10) + "s")
	if err != nil {
		return c.Context.Describe(metric)
	}

	return fmt.Sprintf("running since %s", nagocheck.DurationString(uptimeDuration))
}

func (c *uptimeContext) Evaluate(metric nagopher.Metric, resource nagopher.Resource) nagopher.Result {
	numericMetric, ok := metric.(nagopher.NumericMetric)
	if !ok {
		return nagocheck.NewInvalidMetricTypeResult(c, metric, resource)
	}

	emptyBounds := nagopher.NewBounds()
	warningThreshold := c.warningThreshold.OrElse(emptyBounds)
	criticalThreshold := c.criticalThreshold.OrElse(emptyBounds)

	if !criticalThreshold.Match(numericMetric.Value()) {
		return nagopher.NewResult(
			nagopher.ResultState(nagopher.StateCritical()),
			nagopher.ResultMetric(metric), nagopher.ResultContext(c), nagopher.ResultResource(resource),
			nagopher.ResultHint(c.violationHint(criticalThreshold)),
		)
	} else if !warningThreshold.Match(numericMetric.Value()) {
		return nagopher.NewResult(
			nagopher.ResultState(nagopher.StateWarning()),
			nagopher.ResultMetric(metric), nagopher.ResultContext(c), nagopher.ResultResource(resource),
			nagopher.ResultHint(c.violationHint(warningThreshold)),
		)
	}

	return nagopher.NewResult(
		nagopher.ResultState(nagopher.StateOk()),
		nagopher.ResultMetric(metric), nagopher.ResultContext(c), nagopher.ResultResource(resource),
	)
}

func (c *uptimeContext) violationHint(threshold nagopher.Bounds) string {
	upperBounds := threshold.Upper().OrElse(math.NaN())
	lowerBounds := threshold.Lower().OrElse(math.NaN())

	if math.IsInf(upperBounds, 1) && !math.IsNaN(lowerBounds) {
		boundsDuration, err := time.ParseDuration(strconv.FormatInt(int64(lowerBounds), 10) + "s")
		if err == nil {
			return fmt.Sprintf("less than %s", nagocheck.DurationString(boundsDuration))
		}
	}

	return threshold.ViolationHint()
}

func newUptimeSummarizer(plugin *uptimePlugin) *uptimeSummarizer {
	return &uptimeSummarizer{
		Summarizer: nagocheck.NewSummarizer(plugin),
	}
}
