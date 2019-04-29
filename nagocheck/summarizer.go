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
