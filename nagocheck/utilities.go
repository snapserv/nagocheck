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

package nagocheck

import (
	"fmt"
	"github.com/snapserv/nagopher"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"time"
)

// Round is a utility function which allows rounding a float64 to a given precision
func Round(value float64, precision float64) float64 {
	precision = 1 / math.Pow(10, precision)
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
	daysDuration := compatTimeTruncate(duration, 24*time.Hour)
	days := int64(daysDuration.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd%s", days, (duration - daysDuration).String())
	}

	return duration.Truncate(time.Second).String()
}

// compatTimeTruncate is a compatibility function for time.Truncate(), which is not supported in Go 1.8
func compatTimeTruncate(d time.Duration, m time.Duration) time.Duration {
	if d <= 0 {
		return d
	}

	return d - d%m
}

// FormatBinarySize expects a size given in bytes and returns a formatted string with a precision of two with the most
// appropriate unit, which can either be B, K, M, G or T.
func FormatBinarySize(size float64) string {
	units := []struct {
		Divisor float64
		Suffix  string
	}{
		{math.Pow(1024, 4), "T"},
		{math.Pow(1024, 3), "G"},
		{math.Pow(1024, 2), "M"},
		{math.Pow(1024, 1), "K"},
		{math.Pow(1024, 0), "B"},
	}

	if !math.IsNaN(size) {
		for _, unit := range units {
			if size > unit.Divisor*100 {
				value := Round(size/unit.Divisor, 2)
				return strconv.FormatFloat(value, 'f', 2, strconv.IntSize) + unit.Suffix
			}
		}

		if size == 0 {
			return "0B"
		}
	}

	return "N/A"
}

// NewInvalidMetricTypeResult returns a new Nagopher result in case a custom context tries to convert the generic Metric
// interface pointer into a specific type and is unable to do so. An example from Nagopher itself (which does not use
// this helper method though, obviously) would be a ScalarContext which strictly requires a NumericMetric to properly
// evaluate the context.
func NewInvalidMetricTypeResult(context Context, metric nagopher.Metric, resource nagopher.Resource) nagopher.Result {
	return nagopher.NewResult(
		nagopher.ResultState(nagopher.StateUnknown()),
		nagopher.ResultMetric(metric), nagopher.ResultContext(context), nagopher.ResultResource(resource),
		nagopher.ResultHint(fmt.Sprintf(
			"%s can not process metric of type [%s]", reflect.TypeOf(context), reflect.TypeOf(metric),
		)),
	)
}

type hiddenScalarContext struct {
	Context
}

// NewHiddenScalarContext is a subclass of the standard ScalarContext provided by nagopher. It behaves exactly the same
// in terms of representation and evaluation, however it is being suppressed in performance data.
func NewHiddenScalarContext(plugin Plugin, name string, warningThreshold *nagopher.Bounds, criticalThreshold *nagopher.Bounds) Context {
	return &hiddenScalarContext{
		Context: NewContext(plugin, nagopher.NewScalarContext(
			name, warningThreshold, criticalThreshold,
		)),
	}
}

func (c *hiddenScalarContext) Performance(metric nagopher.Metric, resource nagopher.Resource) (nagopher.OptionalPerfData, error) {
	return nagopher.OptionalPerfData{}, nil
}
