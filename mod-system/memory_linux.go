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
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
)

var meminfoRegexp = regexp.MustCompile(`^(?P<key>\S*):\s*(?P<value>\d*)\s*kB$`)

func (r *memoryResource) Collect() error {
	if err := r.collectMemoryUsage(); err != nil {
		return err
	}

	freeMemory := r.usageStats.free
	if !r.ThisPlugin().CountReclaimable {
		freeMemory += r.usageStats.cached + r.usageStats.buffers
	}
	r.usagePercentage = nagocheck.Round(100-(freeMemory/r.usageStats.total*100), 2)

	return nil
}

func (r *memoryResource) collectMemoryUsage() error {
	bytes, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return fmt.Errorf("could not read memory usage (%s)", err.Error())
	}

	stats := make(map[string]float64)
	lines := strings.Split(string(bytes), "\n")
	for _, line := range lines {
		matchMap, matched := nagocheck.RegexpSubMatchMap(meminfoRegexp, line)
		if !matched {
			continue
		}

		value, err := strconv.ParseFloat(matchMap["value"], strconv.IntSize)
		if err != nil {
			return fmt.Errorf("could not parse [%s] as float (%s)", matchMap["value"], err.Error())
		}

		stats[matchMap["key"]] = value * 1024
	}

	r.usageStats.active = stats["Active"]
	r.usageStats.buffers = stats["Buffers"]
	r.usageStats.cached = stats["Cached"]
	r.usageStats.free = stats["MemFree"]
	r.usageStats.inactive = stats["Inactive"]
	r.usageStats.total = stats["MemTotal"]

	return nil
}
