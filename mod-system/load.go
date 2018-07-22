/*
 * nagocheck - Reliable and lightweight Nagios plugins written in Go
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
	"github.com/snapserv/nagocheck/shared"
	"math"
	"runtime"
)

type loadPlugin struct {
	*shared.BasePlugin

	PerCPU bool
}

type loadSummary struct {
	*shared.BasePluginSummary
}

func newLoadPlugin() *loadPlugin {
	return &loadPlugin{
		BasePlugin: shared.NewPlugin(),
	}
}

func (p *loadPlugin) DefineFlags(kp shared.KingpinInterface) {
	p.BasePlugin.DefineFlags(kp, true)

	kp.Flag("per-cpu", "Enable per-cpu metrics (divide load average by cpu count).").BoolVar(&p.PerCPU)
}

func (p *loadPlugin) Execute() {
	check := nagopher.NewCheck("load", newLoadSummary())
	check.SetMeta(shared.MetaNcPlugin, p)
	check.AttachResources(shared.NewPluginResource(p))
	check.AttachContexts(
		nagopher.NewScalarContext(
			"load",
			p.WarningRange,
			p.CriticalRange,
		),
	)

	p.ExecuteCheck(check)
}

func (p *loadPlugin) Probe(warnings *nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	valueRange, err := nagopher.ParseRange("0:")
	if err != nil {
		return metrics, err
	}

	loadAverages, err := getLoadAverages()
	if err != nil {
		return metrics, err
	}

	metricNames := []string{"load1", "load5", "load15"}
	cpuCount := runtime.NumCPU()

	for key, loadAverage := range loadAverages {
		if p.PerCPU {
			loadAverage /= float64(cpuCount)
		}

		metrics = append(metrics, nagopher.NewNumericMetric(
			metricNames[key], loadAverage, "", valueRange, "load",
		))
	}

	return metrics, nil
}

func newLoadSummary() *loadSummary {
	return &loadSummary{
		BasePluginSummary: shared.NewPluginSummary(),
	}
}

func (s *loadSummary) Ok(check *nagopher.Check) string {
	resultCollection := check.Results()

	return fmt.Sprintf(
		"Load averages%s: %.2f, %.2f, %.2f",

		s.getDescriptionSuffix(check),
		shared.Round(s.GetNumericMetricValue(resultCollection, "load1", math.NaN()), 2),
		shared.Round(s.GetNumericMetricValue(resultCollection, "load5", math.NaN()), 2),
		shared.Round(s.GetNumericMetricValue(resultCollection, "load15", math.NaN()), 2),
	)
}

func (s *loadSummary) Problem(check *nagopher.Check) string {
	resultCollection := check.Results()
	mostSignificantResult := resultCollection.MostSignificantResult()
	if mostSignificantResult == nil {
		return s.BaseSummary.Problem(check)
	}

	metric := mostSignificantResult.Metric()
	metricDescription := map[string]string{
		"load1":  "Load average of last minute",
		"load5":  "Load average of last 5 minutes",
		"load15": "Load average of last 15 minutes",
	}[metric.Name()]

	return fmt.Sprintf("%s%s is %s (%s)",
		metricDescription, s.getDescriptionSuffix(check), metric.ValueString(), mostSignificantResult.Hint())
}

func (s *loadSummary) getDescriptionSuffix(check *nagopher.Check) string {
	if plugin := s.getPlugin(check); plugin != nil {
		if plugin.PerCPU {
			return " per CPU"
		}
	}

	return ""
}

func (s *loadSummary) getPlugin(check *nagopher.Check) *loadPlugin {
	rawPlugin := check.GetMeta(shared.MetaNcPlugin, nil)
	if rawPlugin != nil {
		if plugin, ok := rawPlugin.(*loadPlugin); ok {
			return plugin
		}
	}

	return nil
}
