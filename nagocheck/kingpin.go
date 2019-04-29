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
