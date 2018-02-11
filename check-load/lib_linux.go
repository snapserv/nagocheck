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
	"io/ioutil"
	"strconv"
	"strings"
)

func getLoadAverages() (loadAverages [3]float64, _ error) {
	bytes, err := ioutil.ReadFile("/proc/loadavg")
	if err != nil {
		return loadAverages, fmt.Errorf("load: could not read /proc/loadavg file (%s)", err.Error())
	}

	values := strings.Split(string(bytes), " ")
	for i := 0; i < 3; i++ {
		value, err := strconv.ParseFloat(values[i], strconv.IntSize)
		if err != nil {
			return loadAverages, fmt.Errorf("load: could not parse [%s] as float (%s)", values[i], err.Error())
		}

		loadAverages[i] = value
	}

	return loadAverages, nil
}
