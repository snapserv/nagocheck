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

package shared

import "regexp"

// Round is a utility function which allows rounding a float64 to a given precision
func Round(value float64, precision float64) float64 {
	if value > 0 {
		return float64(int64(value/precision+0.5)) * precision
	}

	return float64(int64(value/precision-0.5)) * precision
}

// RegexpSubMatchMap is a utility function which matches a string against a regular expression and returns a map of the
// type 'map[string]string', which contains all named capture groups.
func RegexpSubMatchMap(r *regexp.Regexp, str string) (map[string]string, bool) {
	subMatchMap := make(map[string]string)

	match := r.FindStringSubmatch(str)
	if match == nil {
		return subMatchMap, false
	}

	for i, name := range r.SubexpNames() {
		if i != 0 && i < len(match) && name != "" {
			subMatchMap[name] = match[i]
		}
	}

	return subMatchMap, true
}
