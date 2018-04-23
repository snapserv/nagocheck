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
	"runtime"
)

type loadPlugin struct {
	*shared.BasePlugin

	PerCPU bool
}

type loadSummary struct {
	*nagopher.BaseSummary
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
		BaseSummary: nagopher.NewBaseSummary(),
	}
}

func (s *loadSummary) Ok(resultCollection *nagopher.ResultCollection) string {
	return fmt.Sprintf(
		"Load averages: %.2f, %.2f, %.2f",

		shared.Round(s.GetNumericMetricValue(resultCollection, "load1", math.NaN()), 2),
		shared.Round(s.GetNumericMetricValue(resultCollection, "load5", math.NaN()), 2),
		shared.Round(s.GetNumericMetricValue(resultCollection, "load15", math.NaN()), 2),
	)
}

func (s *loadSummary) Problem(resultCollection *nagopher.ResultCollection) string {
	result := resultCollection.MostSignificantResult()
	if result == nil {
		return s.BaseSummary.Problem(resultCollection)
	}

	metric := result.Metric()
	metricDescription := map[string]string{
		"load1":  "Load average of last minute",
		"load5":  "Load average of last 5 minutes",
		"load15": "Load average of last 15 minutes",
	}[metric.Name()]

	return fmt.Sprintf("%s is %s (%s)", metricDescription, metric.ValueString(), result.Hint())
}
