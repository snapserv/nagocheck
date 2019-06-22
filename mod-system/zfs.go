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
)

type zfsPlugin struct {
	nagocheck.Plugin
}

type zfsResource struct {
	nagocheck.Resource

	globalStats zfsGlobalStats
	poolStats   map[string]zfsPoolStats
}

type zfsSummarizer struct {
	nagocheck.Summarizer
}

type zfsGlobalStats struct {
	arcSize   uint64
	arcHits   uint64
	arcMisses uint64
}

type zfsPoolStats struct {
	state string
	io    zfsPoolIOStats
}

type zfsPoolIOStats struct {
	readCount    uint64
	writeCount   uint64
	bytesRead    uint64
	bytesWritten uint64
}

func newZfsPlugin() *zfsPlugin {
	return &zfsPlugin{
		Plugin: nagocheck.NewPlugin("zfs",
			nagocheck.PluginDescription("ZFS Pool Statistics"),
			nagocheck.PluginForceVerbose(true),
		),
	}
}

func (p *zfsPlugin) DefineCheck() nagopher.Check {
	check := nagopher.NewCheck("zfs", newZfsSummarizer(p))
	check.AttachResources(newZfsResource(p))
	check.AttachContexts(
		nagopher.NewScalarContext("arc_size", nil, nil),
		nagopher.NewScalarContext("arc_hits", nil, nil),
		nagopher.NewScalarContext("arc_misses", nil, nil),

		nagopher.NewStringMatchContext("pool_state", nagopher.StateCritical(), []string{"ONLINE"}),
		nagopher.NewStringInfoContext("pool"),
	)

	return check
}

func newZfsResource(plugin *zfsPlugin) *zfsResource {
	return &zfsResource{
		Resource: nagocheck.NewResource(plugin),
	}
}

func (r *zfsResource) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	if err := r.Collect(warnings); err != nil {
		return metrics, err
	}

	metrics = append(metrics,
		nagopher.MustNewNumericMetric("arc_size", float64(r.globalStats.arcSize), "B", nil, ""),
		nagopher.MustNewNumericMetric("arc_hits", float64(r.globalStats.arcHits), "c", nil, ""),
		nagopher.MustNewNumericMetric("arc_misses", float64(r.globalStats.arcMisses), "c", nil, ""),
	)

	for poolName, pool := range r.poolStats {
		metrics = append(metrics,
			nagopher.MustNewStringMetric(fmt.Sprintf("pool_%s_state", poolName), pool.state, "pool_state"),
			nagopher.MustNewStringMetric(
				fmt.Sprintf("pool_%s", poolName),
				fmt.Sprintf("%s is %s - %s read, %s written",
					poolName, pool.state,
					nagocheck.FormatBinarySize(float64(pool.io.bytesRead)),
					nagocheck.FormatBinarySize(float64(pool.io.bytesWritten)),
				),
				"pool",
			),
		)
	}

	return metrics, nil
}

func newZfsSummarizer(plugin *zfsPlugin) *zfsSummarizer {
	return &zfsSummarizer{
		Summarizer: nagocheck.NewSummarizer(plugin),
	}
}

func (s *zfsSummarizer) Ok(check nagopher.Check) string {
	resultCollection := check.Results().Get()
	poolCount := 0
	for _, result := range resultCollection {
		context := result.Context().OrElse(nil)
		if context.Name() == "pool_state" {
			poolCount++
		}
	}

	if poolCount == 1 {
		return fmt.Sprintf("%d pool healthy", poolCount)
	}

	return fmt.Sprintf("%d pools healthy", poolCount)
}
