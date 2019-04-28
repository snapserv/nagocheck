package nagocheck

import "github.com/snapserv/nagopher"

type Summarizer interface {
	nagopher.Summarizer
	Plugin() Plugin
}

type SummarizerOpt func(*baseSummarizer)

type baseSummarizer struct {
	nagopher.Summarizer
	plugin Plugin
}

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
