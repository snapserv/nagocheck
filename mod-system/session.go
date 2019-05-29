package modsystem

import (
	"fmt"
	"github.com/shirou/gopsutil/host"
	"github.com/snapserv/nagocheck/nagocheck"
	"github.com/snapserv/nagopher"
	"time"
)

type sessionPlugin struct {
	nagocheck.Plugin

	lifetimeThreshold nagopher.OptionalBounds
}

type sessionResource struct {
	nagocheck.Resource

	sessions []sessionStats
}

type sessionStats struct {
	user     string
	host     string
	terminal string
	lifetime time.Duration
}

type sessionSummarizer struct {
	nagocheck.Summarizer
}

func newSessionPlugin() *sessionPlugin {
	return &sessionPlugin{
		Plugin: nagocheck.NewPlugin("session",
			nagocheck.PluginDescription("User Sessions"),
			nagocheck.PluginForceVerbose(true),
		),
	}
}

func (p *sessionPlugin) DefineFlags(node nagocheck.KingpinNode) {
	nagocheck.NagopherBoundsVar(node.Flag("lifetime", "Lifetime warning threshold formatted as Nagios range specifier.").
		Short('l'), &p.lifetimeThreshold)
}

func (p *sessionPlugin) DefineCheck() nagopher.Check {
	check := nagopher.NewCheck("session", newSessionSummarizer(p))
	check.AttachResources(newSessionResource(p))
	check.AttachContexts(
		nagopher.NewScalarContext(
			"active",
			nagopher.OptionalBoundsPtr(p.WarningThreshold()),
			nagopher.OptionalBoundsPtr(p.CriticalThreshold()),
		),

		nagopher.NewStringInfoContext("session"),
		nagocheck.NewHiddenScalarContext(p, "lifetime", nagopher.OptionalBoundsPtr(p.lifetimeThreshold), nil),
	)

	return check
}

func newSessionResource(plugin *sessionPlugin) *sessionResource {
	return &sessionResource{
		Resource: nagocheck.NewResource(plugin),
	}
}

func (r *sessionResource) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	valueRange := nagopher.NewBounds(nagopher.BoundsOpt(nagopher.LowerBound(0)))

	if err := r.Collect(); err != nil {
		return metrics, err
	}

	metrics = append(metrics,
		nagopher.MustNewNumericMetric("active", float64(len(r.sessions)), "", &valueRange, ""),
	)

	for sessionID, session := range r.sessions {
		metrics = append(metrics,
			nagopher.MustNewStringMetric(
				fmt.Sprintf("session%d", sessionID),
				fmt.Sprintf("#%d %s@%s:%s since %s",
					sessionID, session.user, session.host, session.terminal,
					nagocheck.DurationString(session.lifetime),
				),
				"session",
			),

			nagopher.MustNewNumericMetric(
				fmt.Sprintf("lifetime%d", sessionID),
				float64(session.lifetime.Seconds()), "s", &valueRange, "lifetime",
			),
		)
	}

	return metrics, nil
}

func (r *sessionResource) Collect() error {
	users, err := host.Users()
	if err != nil {
		return err
	}

	r.sessions = make([]sessionStats, 0, len(users))
	for _, user := range users {
		r.sessions = append(r.sessions, sessionStats{
			user:     user.User,
			host:     user.Host,
			terminal: user.Terminal,
			lifetime: time.Now().Sub(time.Unix(int64(user.Started), 0)),
		})
	}

	return nil
}

func newSessionSummarizer(plugin *sessionPlugin) *sessionSummarizer {
	return &sessionSummarizer{
		Summarizer: nagocheck.NewSummarizer(plugin),
	}
}

func (s *sessionSummarizer) Ok(check nagopher.Check) string {
	resultCollection := check.Results()

	return fmt.Sprintf(
		"%d active users",
		int64(resultCollection.GetNumericMetricValue("active").OrElse(0)),
	)
}
