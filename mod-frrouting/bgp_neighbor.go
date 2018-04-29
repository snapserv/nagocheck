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

package modfrrouting

import (
	"encoding/json"
	"fmt"
	"github.com/snapserv/nagopher"
	"github.com/snapserv/nagopher-checks/mod-frrouting/goffr"
	"github.com/snapserv/nagopher-checks/shared"
	"math"
	"strings"
	"time"
)

type bgpNeighborPlugin struct {
	*shared.BasePlugin

	NeighborAddress  string
	IsCritical       bool
	PrefixLimitRange *nagopher.Range

	module *frroutingModule
}

type bgpNeighborSummary struct {
	*shared.BasePluginSummary
}

type bgpNeighborStatsCollection map[string]bgpNeighborStats

type bgpNeighborStats struct {
	LocalHost          string                        `json:"hostLocal"`
	LocalPort          int                           `json:"portLocal"`
	LocalASN           int                           `json:"localAs"`
	RemoteHost         string                        `json:"hostForeign"`
	RemotePort         int                           `json:"portForeign"`
	RemoteASN          int                           `json:"remoteAs"`
	RemoteRouterID     string                        `json:"remoteRouterId"`
	UpdateSource       string                        `json:"updateSource"`
	Version            int                           `json:"bgpVersion"`
	OperationalState   string                        `json:"bgpState"`
	Description        string                        `json:"nbrDesc"`
	AddressFamilies    map[string]bgpNeighborAFStats `json:"addressFamilyInfo"`
	UpTimer            int64                         `json:"bgpTimerUpEstablishedEpoch"`
	ResetTimer         int64                         `json:"lastResetTimerMsecs"`
	ResetReason        string                        `json:"lastResetDueTo"`
	NotificationReason string                        `json:"lastNotificationReason"`

	lastStateChange         time.Duration
	prefixUsageTotal        int
	prefixLimitTotal        int
	prefixLimitUsagePercent int
}

type bgpNeighborAFStats struct {
	PeerGroup   string `json:"peerGroupMember"`
	PrefixCount int    `json:"acceptedPrefixCounter"`
	PrefixLimit int    `json:"prefixAllowedMax"`
}

func newBgpNeighborPlugin(module *frroutingModule) *bgpNeighborPlugin {
	return &bgpNeighborPlugin{
		BasePlugin: shared.NewPlugin(),
		module:     module,
	}
}

func (p *bgpNeighborPlugin) DefineFlags(kp shared.KingpinInterface) {
	p.BasePlugin.DefineFlags(kp, false)

	kp.Flag("neighbor-address", "Specifies the address of neighbor for which the statistics should be "+
		"fetched. Both IPv4 and IPv6 are supported without specifying the address family explicitly.").
		Short('n').
		Required().
		StringVar(&p.NeighborAddress)

	kp.Flag("critical", "Toggles if the given neighbor is critical or not. This will influence the "+
		"resulting check state if the session of the given neighbor is not established by either returning WARNING or "+
		"CRITICAL as the result.").
		Short('c').
		BoolVar(&p.IsCritical)

	shared.NagopherRangeVar(
		kp.Flag("prefix-limit", "Range for prefix limit usage given as Nagios range specifier. Plugin "+
			"will return WARNING state in case the range does not match. If no prefix limit was configured, this "+
			" check gets ignored.").
			Short('l'), &p.PrefixLimitRange)
}

func (p *bgpNeighborPlugin) Execute() {
	problemState := nagopher.StateWarning
	if p.IsCritical {
		problemState = nagopher.StateCritical
	}

	check := nagopher.NewCheck("bgp_neighbor", newBgpNeighborSummary())
	check.AttachResources(shared.NewPluginResource(p))
	check.AttachContexts(
		nagopher.NewStringInfoContext("info_description"),
		nagopher.NewStringInfoContext("info_session_1"),
		nagopher.NewStringInfoContext("info_session_2"),
		nagopher.NewStringInfoContext("info_prefix_usage"),

		nagopher.NewStringMatchContext("state", []string{"ESTABLISHED"}, problemState),
		nagopher.NewScalarContext("last_state_change", nil, nil),
		nagopher.NewScalarContext("prefix_limit_usage", p.PrefixLimitRange, nil),
		nagopher.NewScalarContext("prefix_count", nil, nil),

		nagopher.NewContext("notification_reason", ""),
		nagopher.NewContext("reset_reason", ""),
	)

	p.ExecuteCheck(check)
}

func (p *bgpNeighborPlugin) Probe(warnings *nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	neighbor, err := p.fetchStatistics()
	if err != nil {
		return metrics, err
	}

	lastStateChangeSeconds := shared.Round(neighbor.lastStateChange.Seconds(), 0)
	metrics = append(metrics,
		nagopher.NewStringMetric("state", neighbor.OperationalState, ""),
		nagopher.NewNumericMetric("last_state_change", lastStateChangeSeconds,
			"s", nil, ""),
		nagopher.NewNumericMetric("prefix_count", float64(neighbor.prefixUsageTotal),
			"", nil, ""),

		nagopher.NewStringMetric("reset_reason", neighbor.ResetReason, ""),
		nagopher.NewStringMetric("notification_reason", neighbor.NotificationReason, ""),

		nagopher.NewStringMetric("info_description",
			fmt.Sprintf("description: %s", neighbor.Description),
			""),

		nagopher.NewStringMetric("info_session_1", fmt.Sprintf(
			"session: AS%d[%s:%d] <-> AS%d[%s:%d]",
			neighbor.RemoteASN, neighbor.RemoteHost, neighbor.RemotePort,
			neighbor.LocalASN, neighbor.LocalHost, neighbor.LocalPort),
			""),

		nagopher.NewStringMetric("info_session_2", fmt.Sprintf(
			"session: Version=%d RemoteRID=%s",
			neighbor.Version, neighbor.RemoteRouterID),
			""),
	)

	// Only add prefix limit usage statistics if a prefix limit was set
	if neighbor.prefixLimitTotal > 0 {
		metrics = append(metrics, nagopher.NewNumericMetric("prefix_limit_usage",
			float64(neighbor.prefixLimitUsagePercent), "%", nil, ""))
	}

	// Display additional information about prefix usage
	usageString := fmt.Sprintf("prefixes: %d accepted", neighbor.prefixUsageTotal)
	if neighbor.prefixLimitTotal > 0 {
		usageString += fmt.Sprintf(", %d maximum", neighbor.prefixLimitTotal)
	} else {
		usageString += ", no maximum set"
	}
	metrics = append(metrics, nagopher.NewStringMetric("info_prefix_usage", usageString, ""))

	return metrics, nil
}

func (p *bgpNeighborPlugin) fetchStatistics() (*bgpNeighborStats, error) {
	var neighbors bgpNeighborStatsCollection

	// Establish new goffr instance to the FRRouting daemon
	bgpd, err := p.module.GoffrSession.GetInstance(goffr.InstanceBGP)
	if err != nil {
		return nil, fmt.Errorf("bgp_neighbor: could not connect to bgpd instance (%s)", err.Error())
	}

	// Fetch JSON data for the desired neighbor
	rawData, err := bgpd.ExecuteJSON(fmt.Sprintf("show bgp neighbor %s json", p.NeighborAddress))
	if err != nil {
		return nil, fmt.Errorf("bgp_neighbor: could not fetch statistics for neighbor [%s] (%s)",
			p.NeighborAddress, err.Error())
	}

	// Unmarshal the JSON data into our neighbor statistics struct
	json.Unmarshal([]byte(rawData), &neighbors)
	neighbor, ok := neighbors[strings.ToLower(p.NeighborAddress)]
	if !ok {
		return nil, fmt.Errorf("bgp_neighbor: neighbor [%s] not found", p.NeighborAddress)
	}

	// Manually adjust some returned metrics and/or provide fallback values
	neighbor.OperationalState = strings.ToUpper(neighbor.OperationalState)
	if neighbor.LocalHost == "" {
		neighbor.LocalHost = neighbor.UpdateSource
	}
	if neighbor.RemoteHost == "" {
		neighbor.RemoteHost = p.NeighborAddress
	}

	// Parse FRR state timers (up since OR reset since) to receive 'time.Duration' objects
	if neighbor.UpTimer > 0 {
		// FIXME: Implement 'bgpTimerUpMsec' instead of 'bgpTimerUpEstablishedEpoch' as soon as FRRv4 gets released.
		// This must also be changed within the JSON which is being used for JSON unmarshalling.
		// See: https://github.com/FRRouting/frr/pull/1586/commits/d3c7efede79f88c978efadd850034d472e02cfdb
		neighbor.lastStateChange = time.Since(time.Unix(neighbor.UpTimer, 0))
	} else {
		resetTimer := fmt.Sprintf("%dms", neighbor.ResetTimer)
		neighbor.lastStateChange, err = time.ParseDuration(resetTimer)

		if err != nil {
			return nil, fmt.Errorf("bgp_neighbor: could not parse reset timer [%s] (%s)",
				resetTimer, err.Error())
		}
	}

	// Calculate prefix limit usage in percent for all address families
	for _, addressFamily := range neighbor.AddressFamilies {
		neighbor.prefixUsageTotal += addressFamily.PrefixCount
		neighbor.prefixLimitTotal += addressFamily.PrefixLimit
	}
	if neighbor.prefixLimitTotal > 0 {
		neighbor.prefixLimitUsagePercent = int(float64(neighbor.prefixUsageTotal) / float64(neighbor.prefixLimitTotal) * 100)
	}

	return &neighbor, nil
}

func newBgpNeighborSummary() *bgpNeighborSummary {
	return &bgpNeighborSummary{
		BasePluginSummary: shared.NewPluginSummary(),
	}
}

func (s *bgpNeighborSummary) Ok(check *nagopher.Check) string {
	resultCollection := check.Results()

	lastStateChange := s.GetNumericMetricValue(resultCollection, "last_state_change", math.NaN())
	lastStateChangeString := "N/A"
	if !math.IsNaN(lastStateChange) {
		if lastStateChange > 0 {
			duration, err := time.ParseDuration(fmt.Sprintf("%ds", int(lastStateChange)))
			if err == nil {
				lastStateChangeString = shared.DurationString(duration)
			}
		} else {
			lastStateChangeString = "always"
		}
	}

	return fmt.Sprintf(
		"state is %s since %s",
		s.GetStringMetricValue(resultCollection, "state", "N/A"),
		lastStateChangeString,
	)
}

func (s *bgpNeighborSummary) Problem(check *nagopher.Check) string {
	return s.Ok(check)
}
