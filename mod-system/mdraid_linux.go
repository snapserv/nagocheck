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
	"github.com/snapserv/nagopher"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
)

var personalityNameRE = regexp.MustCompile(`^(linear|raid[0-9]+)$`)
var personalityLineRE = regexp.MustCompile(`(\d+) blocks .*\[(\d+)/(\d+)] \[[u_]+]`)
var personalityRaid0LineRE = regexp.MustCompile(`(\d+) blocks .*\d+k (chunks|rounding)`)
var personalityUnsupportedLineRE = regexp.MustCompile(`(\d+) blocks (.*)`)
var syncLineRE = regexp.MustCompile(`\((\d+)/\d+\)`)

func (r *mdraidResource) Collect(warnings nagopher.WarningCollection) error {
	if err := r.parseMdstat("/proc/mdstat", warnings); err != nil {
		return err
	}

	for i, array := range r.arrays {
		if !array.isActive {
			r.arrays[i].state = "INACTIVE"
		} else if array.blocksSynced != array.blocksTotal {
			r.arrays[i].state = "SYNCING"
		} else {
			r.arrays[i].state = "ACTIVE"
		}
	}

	return nil
}

func (r *mdraidResource) parseMdstat(mdstatPath string, warnings nagopher.WarningCollection) error {
	bytes, err := ioutil.ReadFile(mdstatPath)
	if err != nil {
		return fmt.Errorf("could not read mdstat: %s", err.Error())
	}

	lines := strings.Split(string(bytes), "\n")
	r.arrays = make([]arrayStats, 0, len(lines)/3)
	for index, line := range lines {
		if strings.TrimSpace(line) == "" ||
			line[0] == ' ' || line[0] == '\t' ||
			strings.HasPrefix(line, "Personalities") ||
			strings.HasPrefix(line, "unused") {
			continue
		}

		arrayLine := strings.Split(line, " ")
		if len(arrayLine) < 4 {
			return fmt.Errorf("could not parse invalid mdstat line: %s", arrayLine)
		}

		array := arrayStats{
			name:     arrayLine[0],
			isActive: strings.ToLower(arrayLine[2]) == "active",
		}
		if len(lines) <= index+3 {
			return fmt.Errorf("not enough mdstat lines for array %s", array.name)
		}

		personality := ""
		for _, possiblePersonality := range arrayLine[3:] {
			if personalityNameRE.MatchString(possiblePersonality) {
				personality = strings.ToLower(possiblePersonality)
				break
			}
		}

		personalityLine := strings.ToLower(lines[index+1])
		switch {
		case personality == "raid0" || personality == "linear":
			array.disksActive = uint64(len(arrayLine) - 4)
			array.disksTotal = array.disksActive
			array.blocksTotal, err = r.evaluateRaid0Personality(personalityLine)
		case personalityNameRE.MatchString(personality):
			array.disksActive, array.disksTotal, array.blocksTotal, err = r.evaluatePersonality(personalityLine)
		default:
			warnings.Add(nagopher.NewWarning("unsupported personality: %s", personality))
			array.disksTotal = uint64(len(arrayLine) - 3)
			array.blocksTotal, err = r.evaluateUnsupportedPersonality(personalityLine)
		}

		if err != nil {
			return fmt.Errorf("could not parse mdstat line: %s", err.Error())
		}

		if !array.isActive {
			array.disksActive = 0
		}

		syncLine := strings.ToLower(lines[index+2])
		if strings.Contains(syncLine, "bitmap") {
			syncLine = lines[index+3]
		}

		if strings.Contains(syncLine, "recovery") ||
			strings.Contains(syncLine, "resync") &&
				!strings.Contains(syncLine, "\tresync=") {
			array.blocksSynced, err = r.evaluateSync(syncLine)

			if err != nil {
				return fmt.Errorf("could not parse mdstat line: %s", err.Error())
			}
		} else {
			array.blocksSynced = array.blocksTotal
		}

		r.arrays = append(r.arrays, array)
	}

	return nil
}

func (r *mdraidResource) evaluatePersonality(personalityLine string) (uint64, uint64, uint64, error) {
	matches := personalityLineRE.FindStringSubmatch(personalityLine)

	if len(matches) < 3+1 {
		return 0, 0, 0, fmt.Errorf("too few matches found in personality line: %s", personalityLine)
	} else if len(matches) > 3+1 {
		return 0, 0, 0, fmt.Errorf("too many matches found in personality line: %s", personalityLine)
	}

	blocksTotal, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%s in personality line: %s", err.Error(), personalityLine)
	}

	disksTotal, err := strconv.ParseUint(matches[2], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%s in personality line: %s", err.Error(), personalityLine)
	}

	disksActive, err := strconv.ParseUint(matches[3], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%s in personality line: %s", err.Error(), personalityLine)
	}

	return disksActive, disksTotal, blocksTotal, nil
}

func (r *mdraidResource) evaluateRaid0Personality(personalityLine string) (uint64, error) {
	matches := personalityRaid0LineRE.FindStringSubmatch(personalityLine)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid raid0 personality line: %s", personalityLine)
	}

	blocksTotal, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s in personality line: %s", err.Error(), personalityLine)
	}

	return blocksTotal, nil
}

func (r *mdraidResource) evaluateUnsupportedPersonality(personalityLine string) (uint64, error) {
	matches := personalityUnsupportedLineRE.FindStringSubmatch(personalityLine)
	if len(matches) != 2+1 {
		return 0, fmt.Errorf("invalid unsupported personality line: %s", personalityLine)
	}

	blocksTotal, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s in personality line: %s", err.Error(), personalityLine)
	}

	return blocksTotal, nil
}

func (r *mdraidResource) evaluateSync(syncLine string) (uint64, error) {
	matches := syncLineRE.FindStringSubmatch(syncLine)

	if len(matches) < 1+1 {
		return 0, fmt.Errorf("too few matches found in sync line: %s", syncLine)
	} else if len(matches) > 1+1 {
		return 0, fmt.Errorf("too many matches found in sync line: %s", syncLine)
	}

	blocksSynced, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s in sync line: %s", err, syncLine)
	}

	return blocksSynced, nil
}
