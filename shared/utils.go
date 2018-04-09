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

import (
	"fmt"
	"regexp"
	"time"
)

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

// RetryDuring retries a given function until it no longer returns an error or the timeout value was reached. The delay
// parameter specifies the delay between each unsuccessful attempt.
func RetryDuring(timeout time.Duration, delay time.Duration, function func() error) (err error) {
	startTime := time.Now()
	attempts := 0
	for {
		attempts++

		err = function()
		if err == nil {
			return
		}

		deltaTime := time.Now().Sub(startTime)
		if deltaTime > timeout {
			return fmt.Errorf("aborting retrying after %d attempts (during %s), last error: %s",
				attempts, deltaTime, err.Error())
		}

		time.Sleep(delay)
	}
}

// DurationString outputs a time.Duration variable in the same way as time.Duration.String() with additional support for
// days instead of just hours, minutes and seconds.
func DurationString(duration time.Duration) string {
	daysDuration := duration.Truncate(24 * time.Hour)
	days := int64(daysDuration.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd%s", days, (duration - daysDuration).String())
	}

	return duration.String()
}
