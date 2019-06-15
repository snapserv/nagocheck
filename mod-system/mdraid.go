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
	"strings"
)

type mdraidPlugin struct {
	nagocheck.Plugin
}

type mdraidResource struct {
	nagocheck.Resource

	arrays []arrayStats
}

type mdraidSummarizer struct {
	nagocheck.Summarizer
}

type arrayStats struct {
	state        string
	name         string
	isActive     bool
	disksActive  uint64
	disksTotal   uint64
	blocksSynced uint64
	blocksTotal  uint64
}

func newMdraidPlugin() *mdraidPlugin {
	return &mdraidPlugin{
		Plugin: nagocheck.NewPlugin("mdraid",
			nagocheck.PluginDescription("MD RAID"),
			nagocheck.PluginForceVerbose(true),
		),
	}
}

func (p *mdraidPlugin) DefineCheck() nagopher.Check {
	check := nagopher.NewCheck("mdraid", newMdraidSummarizer(p))
	check.AttachResources(newMdraidResource(p))
	check.AttachContexts(
		nagopher.NewStringMatchContext("state", nagopher.StateCritical(), []string{"ACTIVE"}),
		nagopher.NewStringInfoContext("array"),

		nagopher.NewScalarContext("disks_active", nil, nil),
		nagopher.NewScalarContext("disks_total", nil, nil),
		nagopher.NewScalarContext("blocks_synced", nil, nil),
		nagopher.NewScalarContext("blocks_total", nil, nil),
	)

	return check
}

func newMdraidResource(plugin *mdraidPlugin) *mdraidResource {
	return &mdraidResource{
		Resource: nagocheck.NewResource(plugin),
	}
}

func (r *mdraidResource) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	if err := r.Collect(warnings); err != nil {
		return metrics, err
	}

	if len(r.arrays) == 0 {
		return metrics, fmt.Errorf("no arrays available")
	}

	for _, array := range r.arrays {
		metrics = append(metrics,
			nagopher.MustNewStringMetric(array.name+"_state", array.state, "state"),
			nagopher.MustNewStringMetric(array.name+"_array",
				fmt.Sprintf("%s: %s with %d/%d disks and %d blocks",
					array.name, strings.ToLower(array.state),
					array.disksActive, array.disksTotal, array.blocksTotal,
				),
				"array",
			),

			nagopher.MustNewNumericMetric(array.name+"_disks_active", float64(array.disksActive), "", nil, "disks_active"),
			nagopher.MustNewNumericMetric(array.name+"_disks_total", float64(array.disksTotal), "", nil, "disks_total"),
			nagopher.MustNewNumericMetric(array.name+"_blocks_synced", float64(array.blocksSynced), "", nil, "blocks_synced"),
			nagopher.MustNewNumericMetric(array.name+"_blocks_total", float64(array.blocksTotal), "", nil, "blocks_total"),
		)
	}

	return metrics, nil
}

func newMdraidSummarizer(plugin *mdraidPlugin) *mdraidSummarizer {
	return &mdraidSummarizer{
		Summarizer: nagocheck.NewSummarizer(plugin),
	}
}

func (s *mdraidSummarizer) Ok(check nagopher.Check) string {
	resultCollection := check.Results().Get()
	arrayCount := 0
	for _, result := range resultCollection {
		context := result.Context().OrElse(nil)
		if context.Name() == "state" {
			arrayCount++
		}
	}

	return fmt.Sprintf("%d arrays healthy", arrayCount)
}
