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

// Summarizer provides a base type for nagocheck summarizers, which embeds nagopher.Summarizer
type Summarizer interface {
	nagopher.Summarizer
	Plugin() Plugin
}

// SummarizerOpt is a type alias for functional options used by NewSummarizer()
type SummarizerOpt func(*baseSummarizer)

type baseSummarizer struct {
	nagopher.Summarizer
	plugin Plugin
}

// NewSummarizer instantiates baseSummarizer with the given functional options
func NewSummarizer(plugin Plugin, options ...SummarizerOpt) Summarizer {
	summarizer := &baseSummarizer{
		Summarizer: nagopher.NewSummarizer(),
		plugin:     plugin,
	}

	for _, option := range options {
		option(summarizer)
	}

	return summarizer
}

func (s *baseSummarizer) Plugin() Plugin {
	return s.plugin
}
