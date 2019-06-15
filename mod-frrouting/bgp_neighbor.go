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

package modfrrouting

import (
	"fmt"
	"github.com/snapserv/nagocheck/nagocheck"
	"github.com/snapserv/nagopher"
	"math"
	"net"
	"time"
)

type bgpNeighborPlugin struct {
	nagocheck.Plugin
	myModule *frroutingModule

	NeighborIP       net.IP
	IsCritical       bool
	PrefixLimitRange nagopher.OptionalBounds
	UptimeRange      nagopher.OptionalBounds
}

type bgpNeighborResource struct {
	nagocheck.Resource

	neighbor *BgpNeighbor
}

type bgpNeighborSummarizer struct {
	nagocheck.Summarizer
}

type uptimeContext struct {
	nagopher.Context
}

func newBgpNeighborPlugin() *bgpNeighborPlugin {
	return &bgpNeighborPlugin{
		Plugin: nagocheck.NewPlugin("bgp-neighbor",
			nagocheck.PluginDescription("BGP Neighbor"),
			nagocheck.PluginDefaultThresholds(false),
		),
	}
}

func (p *bgpNeighborPlugin) DefineFlags(node nagocheck.KingpinNode) {
	node.Arg("neighbor", "Specifies the IP address of neighbor for which the statistics should be fetched. Both IPv4 "+
		"IPv6 are supported without specifying the address family explicitly.").
		Required().IPVar(&p.NeighborIP)

	node.Flag("critical", "Toggles if the given neighbor is critical or not. This will influence the "+
		"resulting check state if the session of the given neighbor is not established by either returning WARNING or "+
		"CRITICAL as the result.").
		Short('c').BoolVar(&p.IsCritical)

	nagocheck.NagopherBoundsVar(node.Flag("prefix-limit", "Range for prefix limit usage given as Nagios range specifier. "+
		"Plugin will return WARNING state in case the range does not match. If no prefix limit was configured, this "+
		"check gets ignored.").
		Short('l'), &p.PrefixLimitRange)

	nagocheck.NagopherBoundsVar(node.Flag("uptime", "Range for neighbor uptime (state=ESTABLISHED) given as Nagios range "+
		"specifier. Plugin will return WARNING state in case the range does not match. This allows to alert when a "+
		"session was recently established.").
		Short('u'), &p.UptimeRange)
}

func (p *bgpNeighborPlugin) DefineCheck() nagopher.Check {
	problemState := nagopher.StateWarning()
	if p.IsCritical {
		problemState = nagopher.StateCritical()
	}

	check := nagopher.NewCheck("bgp_neighbor", newBgpNeighborSummarizer(p))
	check.AttachResources(newBgpNeighborResource(p))
	check.AttachContexts(
		nagopher.NewStringInfoContext("info_description"),
		nagopher.NewStringInfoContext("info_session_1"),
		nagopher.NewStringInfoContext("info_session_2"),
		nagopher.NewStringInfoContext("info_prefix_usage"),
		nagopher.NewStringInfoContext("info_reset_reason"),
		nagopher.NewStringInfoContext("info_notification_reason"),

		nagopher.NewStringMatchContext("state", problemState, []string{"ESTABLISHED"}),
		nagopher.NewScalarContext("last_state_change", nil, nil),
		nagopher.NewScalarContext("prefix_limit_usage", nagopher.OptionalBoundsPtr(p.PrefixLimitRange), nil),
		nagopher.NewScalarContext("prefix_count", nil, nil),

		newUptimeContext("uptime", nagopher.OptionalBoundsPtr(p.UptimeRange), nil),
	)

	return check
}

func (p *bgpNeighborPlugin) ThisModule() *frroutingModule {
	return p.Plugin.Module().(*frroutingModule)
}

func newBgpNeighborResource(plugin *bgpNeighborPlugin) *bgpNeighborResource {
	return &bgpNeighborResource{
		Resource: nagocheck.NewResource(plugin),
	}
}

func (r *bgpNeighborResource) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	if err := r.Collect(); err != nil {
		return metrics, err
	}

	lastStateChangeSeconds := nagocheck.Round(r.neighbor.LastStateChange.Seconds(), 0)
	metrics = append(metrics,
		nagopher.MustNewStringMetric("state", r.neighbor.OperationalState, ""),
		nagopher.MustNewNumericMetric("last_state_change", lastStateChangeSeconds, "s", nil, ""),
		nagopher.MustNewNumericMetric("prefix_count", float64(r.neighbor.PrefixUsageTotal), "", nil, ""),

		nagopher.MustNewStringMetric("info_description", fmt.Sprintf(
			"description: %s",
			r.neighbor.Description), ""),
		nagopher.MustNewStringMetric("info_session_1", fmt.Sprintf(
			"session: AS%d[%s:%d] <-> AS%d[%s:%d]",
			r.neighbor.RemoteAS, r.neighbor.RemoteHost, r.neighbor.RemotePort,
			r.neighbor.LocalAS, r.neighbor.LocalHost, r.neighbor.LocalPort), ""),
		nagopher.MustNewStringMetric("info_session_2", fmt.Sprintf(
			"session: Version=%d RemoteRID=%s",
			r.neighbor.Version, r.neighbor.RemoteRouterID), ""),
	)

	// Only add prefix limit usage statistics if a prefix limit was set
	if r.neighbor.PrefixLimitTotal > 0 {
		percentage := float64(r.neighbor.PrefixUsageTotal / r.neighbor.PrefixLimitTotal * 100)
		metrics = append(metrics, nagopher.MustNewNumericMetric("prefix_limit_usage", percentage, "%", nil, ""))
	}

	// Only add uptime metric (redundant with last state change metric) if state=='ESTABLISHED'
	if r.neighbor.OperationalState == "ESTABLISHED" {
		metrics = append(metrics, nagopher.MustNewNumericMetric("uptime", lastStateChangeSeconds, "s", nil, ""))
	}

	// Display additional information about prefix usage
	usageString := fmt.Sprintf("prefixes: %d accepted", r.neighbor.PrefixUsageTotal)
	if r.neighbor.PrefixLimitTotal > 0 {
		usageString += fmt.Sprintf(", %d maximum", r.neighbor.PrefixLimitTotal)
	} else {
		usageString += ", no maximum set"
	}
	metrics = append(metrics, nagopher.MustNewStringMetric("info_prefix_usage", usageString, ""))

	// Display last reset/notification reason if neighbor has state!='ESTABLISHED' and not reason is not empty
	if r.neighbor.OperationalState != "ESTABLISHED" {
		if r.neighbor.ResetReason != "" {
			metrics = append(metrics, nagopher.MustNewStringMetric("info_reset_reason",
				fmt.Sprintf("last reset reason: %s", r.neighbor.ResetReason), ""))
		}
		if r.neighbor.NotificationReason != "" {
			metrics = append(metrics, nagopher.MustNewStringMetric("info_notification_reason",
				fmt.Sprintf("last notification reason: %s", r.neighbor.NotificationReason), ""))
		}
	}

	return metrics, nil
}

func (r *bgpNeighborResource) Collect() error {
	var err error

	r.neighbor, err = r.Session().GetBgpNeighbor(r.ThisPlugin().NeighborIP.String())
	return err
}

func (r *bgpNeighborResource) Session() Session {
	return r.ThisPlugin().ThisModule().session
}

func (r *bgpNeighborResource) ThisPlugin() *bgpNeighborPlugin {
	return r.Resource.Plugin().(*bgpNeighborPlugin)
}

func newBgpNeighborSummarizer(plugin *bgpNeighborPlugin) *bgpNeighborSummarizer {
	return &bgpNeighborSummarizer{
		Summarizer: nagocheck.NewSummarizer(plugin),
	}
}

func (s *bgpNeighborSummarizer) Ok(check nagopher.Check) string {
	resultCollection := check.Results()

	lastStateChange := resultCollection.GetNumericMetricValue("last_state_change").OrElse(math.NaN())
	lastStateChangeString := "N/A"
	if !math.IsNaN(lastStateChange) {
		if lastStateChange > 0 {
			duration, err := time.ParseDuration(fmt.Sprintf("%ds", int(lastStateChange)))
			if err == nil {
				lastStateChangeString = nagocheck.DurationString(duration)
			}
		} else {
			lastStateChangeString = "always"
		}
	}

	return fmt.Sprintf(
		"state is %s since %s",
		resultCollection.GetStringMetricValue("state").OrElse("N/A"),
		lastStateChangeString,
	)
}

func (s *bgpNeighborSummarizer) Problem(check nagopher.Check) string {
	result, err := check.Results().MostSignificantResult().Get()
	if err == nil && result != nil {
		metric, err := result.Metric().Get()
		if err == nil && metric != nil && metric.Name() == "state" {
			return s.Ok(check)
		}
	}

	return s.Summarizer.Problem(check)
}

func newUptimeContext(name string, warningThreshold *nagopher.Bounds, criticalThreshold *nagopher.Bounds) nagopher.Context {
	return &uptimeContext{nagopher.NewScalarContext(name, warningThreshold, criticalThreshold)}
}

func (c *uptimeContext) Performance(metric nagopher.Metric, resource nagopher.Resource) (nagopher.OptionalPerfData, error) {
	return nagopher.OptionalPerfData{}, nil
}
