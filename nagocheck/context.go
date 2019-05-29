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

import "github.com/snapserv/nagopher"

// Context provides a base type for nagocheck contexts, which embeds nagopher.Context
type Context interface {
	nagopher.Context
	Plugin() Plugin
}

// ContextOpt is a type alias for functional options used by NewContext()
type ContextOpt func(*baseContext)

type baseContext struct {
	nagopher.Context
	plugin Plugin
}

// NewContext instantiates baseContext with the given functional options
func NewContext(plugin Plugin, parentContext nagopher.Context, options ...ContextOpt) Context {
	context := &baseContext{
		Context: parentContext,
		plugin:  plugin,
	}

	for _, option := range options {
		option(context)
	}

	return context
}

func (s *baseContext) Plugin() Plugin {
	return s.plugin
}
