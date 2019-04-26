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
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
)

var procLoadavgPath = "/proc/loadavg"

func (s *loadStats) Collect(perCPU bool) error {
	s.cpuCores = uint(runtime.NumCPU())
	if err := s.collectLoadAverages(perCPU); err != nil {
		return err
	}

	return nil
}

func (s *loadStats) collectLoadAverages(perCPU bool) error {
	bytes, err := ioutil.ReadFile(procLoadavgPath)
	if err != nil {
		return fmt.Errorf("could not read load averages from [%s]: %s", procLoadavgPath, err.Error())
	}

	values := strings.Split(string(bytes), " ")
	if len(values) < 3 {
		return fmt.Errorf("could not parse unknown format from [%s]: expected 3 space-separated values", procLoadavgPath)
	}

	loadAverages := make([]float64, 0, 3)
	for i := 0; i < 3; i++ {
		value, err := strconv.ParseFloat(values[i], strconv.IntSize)
		if err != nil {
			return fmt.Errorf("could not parse [%s] as float (%s)", values[i], err.Error())
		}

		if perCPU {
			value /= float64(s.cpuCores)
		}
		loadAverages = append(loadAverages, value)
	}

	s.loadAverage1 = loadAverages[0]
	s.loadAverage5 = loadAverages[1]
	s.loadAverage15 = loadAverages[2]

	return nil
}
