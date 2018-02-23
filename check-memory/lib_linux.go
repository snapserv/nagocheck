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

package main

import (
	"fmt"
	"github.com/snapserv/nagopher-checks/shared"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
)

func getMemoryUsage() (*memoryUsage, error) {
	bytes, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("memory: could not read /proc/meminfo file (%s)", err.Error())
	}

	re, err := regexp.Compile(`^(?P<key>\S*):\s*(?P<value>\d*)\s*kB$`)
	if err != nil {
		return nil, fmt.Errorf("memory: could not compile regular expression (%s)", err.Error())
	}

	stats := make(map[string]float64)
	lines := strings.Split(string(bytes), "\n")
	for _, line := range lines {
		matchMap, matched := shared.RegexpSubMatchMap(re, line)
		if !matched {
			continue
		}

		value, err := strconv.ParseFloat(matchMap["value"], strconv.IntSize)
		if err != nil {
			return nil, fmt.Errorf("memory: could not parse value [%s] as float (%s)",
				matchMap["value"], err.Error())
		}

		stats[matchMap["key"]] = value
	}

	return &memoryUsage{
		active:   stats["Active"],
		buffers:  stats["Buffers"],
		cached:   stats["Cached"],
		free:     stats["MemFree"],
		inactive: stats["Inactive"],
		total:    stats["MemTotal"],
	}, nil
}
