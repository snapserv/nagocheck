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
	"github.com/shirou/gopsutil/load"
	"github.com/snapserv/nagocheck/nagocheck"
	"github.com/snapserv/nagopher"
	"math"
	"runtime"
)

type loadPlugin struct {
	nagocheck.Plugin

	PerCPU bool
}

type loadResource struct {
	nagocheck.Resource

	cpuCores      uint
	loadAverage1  float64
	loadAverage5  float64
	loadAverage15 float64
}

type loadSummarizer struct {
	nagocheck.Summarizer
}

func newLoadPlugin() *loadPlugin {
	return &loadPlugin{
		Plugin: nagocheck.NewPlugin("load",
			nagocheck.PluginDescription("Load Average"),
		),
		PerCPU: false,
	}
}

func (p *loadPlugin) DefineFlags(kp nagocheck.KingpinNode) {
	kp.Flag("per-cpu", "Enable per-cpu metrics (divide load average by cpu count).").BoolVar(&p.PerCPU)
}

func (p *loadPlugin) DefineCheck() nagopher.Check {
	check := nagopher.NewCheck("load", newLoadSummarizer(p))
	check.AttachResources(newLoadResource(p))
	check.AttachContexts(
		nagopher.NewScalarContext(
			"load",
			nagopher.OptionalBoundsPtr(p.WarningThreshold()),
			nagopher.OptionalBoundsPtr(p.CriticalThreshold()),
		),
	)

	return check
}

func newLoadResource(plugin *loadPlugin) *loadResource {
	return &loadResource{
		Resource: nagocheck.NewResource(plugin),
	}
}

func (r *loadResource) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	valueRange := nagopher.NewBounds(nagopher.BoundsOpt(nagopher.LowerBound(0)))

	if err := r.Collect(); err != nil {
		return metrics, err
	}

	metrics = append(metrics,
		nagopher.MustNewNumericMetric("load1", r.loadAverage1, "", &valueRange, "load"),
		nagopher.MustNewNumericMetric("load5", r.loadAverage5, "", &valueRange, "load"),
		nagopher.MustNewNumericMetric("load15", r.loadAverage15, "", &valueRange, "load"),
	)

	return metrics, nil
}

func (r *loadResource) Collect() error {
	loadStats, err := load.Avg()
	if err != nil {
		return err
	}

	r.cpuCores = uint(runtime.NumCPU())
	r.loadAverage1 = loadStats.Load1
	r.loadAverage5 = loadStats.Load5
	r.loadAverage15 = loadStats.Load15

	if r.ThisPlugin().PerCPU {
		r.loadAverage1 /= float64(r.cpuCores)
		r.loadAverage5 /= float64(r.cpuCores)
		r.loadAverage15 /= float64(r.cpuCores)
	}

	return nil
}

func (r *loadResource) ThisPlugin() *loadPlugin {
	return r.Resource.Plugin().(*loadPlugin)
}

func newLoadSummarizer(plugin *loadPlugin) *loadSummarizer {
	return &loadSummarizer{
		Summarizer: nagocheck.NewSummarizer(plugin),
	}
}

func (s *loadSummarizer) Ok(check nagopher.Check) string {
	resultCollection := check.Results()

	return fmt.Sprintf(
		"Load averages%s: %.2f, %.2f, %.2f",

		s.getDescriptionSuffix(check),
		nagocheck.Round(resultCollection.GetNumericMetricValue("load1").OrElse(math.NaN()), 2),
		nagocheck.Round(resultCollection.GetNumericMetricValue("load5").OrElse(math.NaN()), 2),
		nagocheck.Round(resultCollection.GetNumericMetricValue("load15").OrElse(math.NaN()), 2),
	)
}

func (s *loadSummarizer) Problem(check nagopher.Check) string {
	mostSignificantResult, err := check.Results().MostSignificantResult().Get()
	if err != nil || mostSignificantResult == nil {
		return s.Summarizer.Problem(check)
	}

	metric, err := mostSignificantResult.Metric().Get()
	if err != nil || metric == nil {
		return s.Summarizer.Problem(check)
	}

	metricDescription := map[string]string{
		"load1":  "Load average of last minute",
		"load5":  "Load average of last 5 minutes",
		"load15": "Load average of last 15 minutes",
	}[metric.Name()]

	return fmt.Sprintf("%s%s is %s (%s)", metricDescription, s.getDescriptionSuffix(check),
		metric.ValueString(), mostSignificantResult.Hint())
}

func (s *loadSummarizer) getDescriptionSuffix(check nagopher.Check) string {
	if s.ThisPlugin().PerCPU {
		return " per CPU"
	}

	return ""
}

func (s *loadSummarizer) ThisPlugin() *loadPlugin {
	return s.Summarizer.Plugin().(*loadPlugin)
}
