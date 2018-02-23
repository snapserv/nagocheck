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
	"github.com/snapserv/nagopher"
	"gopkg.in/alecthomas/kingpin.v2"
)

type nagopherRangeValue struct{ value *nagopher.Range }

func (r *nagopherRangeValue) Set(rawValue string) error {
	value, err := nagopher.ParseRange(rawValue)
	if err == nil {
		r.value = value
	}

	return err
}

func (r *nagopherRangeValue) String() string {
	return r.value.String()
}

// NagopherRangeVar is a helper method for defining kingpin flags which should be parsed as a Nagopher range specifier.
func NagopherRangeVar(s kingpin.Settings, target *nagopher.Range) {
	s.SetValue(&nagopherRangeValue{target})
}
