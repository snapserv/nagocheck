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

type loadPlugin struct {
	shared.Plugin

	stats  loadStats
	PerCPU bool
}

type loadSummarizer struct {
	shared.PluginSummarizer
}

type loadStats struct {
	cpuCores      uint
	loadAverage1  float64
	loadAverage5  float64
	loadAverage15 float64
}

func newLoadPlugin() *loadPlugin {
	return &loadPlugin{
		Plugin: shared.NewPlugin(),
		PerCPU: false,
	}
}

func (p *loadPlugin) DefineFlags(kp shared.KingpinNode) {
	p.Plugin.DefineDefaultFlags(kp)
	p.Plugin.DefineDefaultThresholds(kp)

	kp.Flag("per-cpu", "Enable per-cpu metrics (divide load average by cpu count).").BoolVar(&p.PerCPU)
}

func (p *loadPlugin) Execute() {
	check := nagopher.NewCheck("load", newLoadSummarizer(p))
	check.SetMeta(shared.MetaNcPlugin, p)
	check.AttachResources(shared.NewPluginResource(p))
	check.AttachContexts(
		nagopher.NewScalarContext(
			"load",
			nagopher.OptionalBoundsPtr(p.WarningThreshold()),
			nagopher.OptionalBoundsPtr(p.CriticalThreshold()),
		),
	)

	p.ExecuteCheck(check)
}

func (p *loadPlugin) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	valueRange, err := nagopher.NewBoundsFromNagiosRange("0:")
	if err != nil {
		return metrics, err
	}

	if err := p.stats.Collect(p.PerCPU); err != nil {
		return metrics, err
	}

	metrics = append(metrics,
		nagopher.MustNewNumericMetric("load1", p.stats.loadAverage1, "", &valueRange, "load"),
		nagopher.MustNewNumericMetric("load5", p.stats.loadAverage5, "", &valueRange, "load"),
		nagopher.MustNewNumericMetric("load15", p.stats.loadAverage15, "", &valueRange, "load"),
	)

	return metrics, nil
}

func newLoadSummarizer(plugin *loadPlugin) *loadSummarizer {
	return &loadSummarizer{
		PluginSummarizer: shared.NewPluginSummarizer(plugin),
	}
}

func (s *loadSummarizer) Ok(check nagopher.Check) string {
	resultCollection := check.Results()

	return fmt.Sprintf(
		"Load averages%s: %.2f, %.2f, %.2f",

		s.getDescriptionSuffix(check),
		shared.Round(resultCollection.GetNumericMetricValue("load1").OrElse(math.NaN()), 2),
		shared.Round(resultCollection.GetNumericMetricValue("load5").OrElse(math.NaN()), 2),
		shared.Round(resultCollection.GetNumericMetricValue("load15").OrElse(math.NaN()), 2),
	)
}

func (s *loadSummarizer) Problem(check nagopher.Check) string {
	mostSignificantResult, err := check.Results().MostSignificantResult().Get()
	if err != nil || mostSignificantResult == nil {
		return s.PluginSummarizer.Problem(check)
	}

	metric, err := mostSignificantResult.Metric().Get()
	if err != nil || metric == nil {
		return s.PluginSummarizer.Problem(check)
	}

	metricDescription := map[string]string{
		"load1":  "Load average of last minute",
		"load5":  "Load average of last 5 minutes",
		"load15": "Load average of last 15 minutes",
	}[metric.Name()]

	return fmt.Sprintf("%s%s is %s (%s)", metricDescription, s.getDescriptionSuffix(check),
		metric.ValueString(), mostSignificantResult.Hint())
}

func (s loadSummarizer) getDescriptionSuffix(check nagopher.Check) string {
	if plugin := s.getPlugin(check); plugin != nil {
		if plugin.PerCPU {
			return " per CPU"
		}
	}

	return ""
}

func (s loadSummarizer) getPlugin(check nagopher.Check) *loadPlugin {
	rawPlugin := check.GetMeta(shared.MetaNcPlugin, nil)
	if rawPlugin != nil {
		if plugin, ok := rawPlugin.(*loadPlugin); ok {
			return plugin
		}
	}

	return nil
}
