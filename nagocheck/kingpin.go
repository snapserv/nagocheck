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
	"github.com/snapserv/nagopher"
	"gopkg.in/alecthomas/kingpin.v2"
)

// KingpinNode is a unified interface for kingpin, which allows using Arg() and Flag() at root- and command-level
type KingpinNode interface {
	Arg(name, help string) *kingpin.ArgClause
	Flag(name, help string) *kingpin.FlagClause
}

type nagopherBoundsValue struct {
	value *nagopher.OptionalBounds
}

func (r *nagopherBoundsValue) Set(rawValue string) error {
	value, err := nagopher.NewBoundsFromNagiosRange(rawValue)
	if err == nil {
		(*r.value).Set(value)
	}

	return err
}

func (r *nagopherBoundsValue) String() string {
	return (*r.value).OrElse(nagopher.NewBounds()).String()
}

// NagopherBoundsVar is a helper method for defining kingpin flags which should be parsed as a Nagopher range specifier.
func NagopherBoundsVar(s kingpin.Settings, target *nagopher.OptionalBounds) {
	s.SetValue(&nagopherBoundsValue{target})
}
