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

package shared

import (
	"fmt"
	"github.com/snapserv/nagopher"
	"github.com/theckman/go-flock"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"syscall"
	"time"
)

// MetaNcPlugin contains the metadata key for storing the plugin instance.
const MetaNcPlugin = "nc#plugin"

// KingpinNode is a generic interface for kingpin, which is implemented by both "kingpin.Application" and
// "kingpin.CmdClause". This allows us to define a single "DefineFlags" method in the "Plugin" interface, which can
// handle both top-level and command-level flags.
type KingpinNode interface {
	Arg(name, help string) *kingpin.ArgClause
	Flag(name, help string) *kingpin.FlagClause
}

// Plugin represents a interface for all plugin types.
type Plugin interface {
	DefineFlags(kp KingpinNode)
	DefineDefaultFlags(kp KingpinNode)
	DefineDefaultThresholds(kp KingpinNode)

	Execute()
	ExecuteCheck(check nagopher.Check)
	ExecutePersistentCheck(check nagopher.Check, uniqueKey string, store interface{})
	Probe(nagopher.WarningCollection) ([]nagopher.Metric, error)

	WarningThreshold() nagopher.OptionalBounds
	CriticalThreshold() nagopher.OptionalBounds
}

// PluginResource represents a resource tied to a plugin
type PluginResource interface {
	nagopher.Resource
	Plugin() Plugin
}

// PluginSummarizer represents a summarizer tied to a plugin
type PluginSummarizer interface {
	nagopher.Summarizer
	Plugin() Plugin
}

type basePlugin struct {
	verbose           bool
	warningThreshold  nagopher.OptionalBounds
	criticalThreshold nagopher.OptionalBounds
}

type pluginResource struct {
	nagopher.Resource
	plugin Plugin
}

type pluginSummarizer struct {
	nagopher.Summarizer
	plugin Plugin
}

func NewPlugin() Plugin {
	return &basePlugin{}
}

func (p *basePlugin) DefineFlags(kp KingpinNode) {
	p.DefineDefaultFlags(kp)
	p.DefineDefaultThresholds(kp)
}

func (p *basePlugin) DefineDefaultFlags(kp KingpinNode) {
	kp.Flag("verbose", "Enable verbose plugin output.").
		Short('v').BoolVar(&p.verbose)
}

func (p *basePlugin) DefineDefaultThresholds(kp KingpinNode) {
	NagopherBoundsVar(kp.Flag("warning", "Warning threshold formatted as Nagios range specifier.").
		Short('w'), &p.warningThreshold)
	NagopherBoundsVar(kp.Flag("critical", "Critical threshold formatted as Nagios range specifier.").
		Short('c'), &p.criticalThreshold)
}

func (p *basePlugin) Execute() {}

func (p basePlugin) WarningThreshold() nagopher.OptionalBounds {
	return p.warningThreshold
}

func (p basePlugin) CriticalThreshold() nagopher.OptionalBounds {
	return p.criticalThreshold
}

// ExecuteCheck is a helper method which creates a new nagopher 'Runtime', executes a check and exits
func (p basePlugin) ExecuteCheck(check nagopher.Check) {
	runtime := nagopher.NewRuntime(p.verbose)
	runtime.ExecuteAndExit(check)
}

// ExecutePersistentCheck is a helper method which extends Execute() with flock (based on given unique key, which should be
// chosen wisely) and a persistent store, which is also named by the unique key passed. This is especially useful when
// used with contexts like 'DeltaContext', which compare the current measurement against a previously measurement.
func (p *basePlugin) ExecutePersistentCheck(check nagopher.Check, uniqueKey string, store interface{}) {
	// Prefix unique key with 'nagocheck.'
	uniqueKey = "nagocheck." + uniqueKey

	// Attempt to grab flock on unique key
	fileLock := p.createFlock(uniqueKey)
	defer fileLock.Unlock()
	if err := p.ensureFlock(fileLock); err != nil {
		panic(err)
	}

	// Load plugin persistence store
	if err := LoadPersistentStore(uniqueKey, store); err != nil {
		panic(err)
	}

	// Execute check with nagopher runtime
	runtime := nagopher.NewRuntime(p.verbose)
	result := runtime.Execute(check)

	// Save plugin persistence store
	if err := SavePersistentStore(uniqueKey, store); err != nil {
		panic(err)
	}

	// Unlink and unlock flock immediately after execution
	syscall.Unlink(fileLock.Path())
	fileLock.Unlock()

	// Print plugin output and exit with the according exit code
	fmt.Print(result.Output())
	os.Exit(int(result.ExitCode()))
}

// Probe represents the method executing the actual check/metrics logic and should be overridden by each plugin for
// returning metrics. It also supports adding warnings through the passed 'WarningCollection' or returning an error in
// case metric collection goes wrong.
func (p *basePlugin) Probe(warnings nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	return metrics, nil
}

func (p *basePlugin) createFlock(identifier string) *flock.Flock {
	return flock.NewFlock(fmt.Sprintf("/tmp/.%s.lock", identifier))
}

func (p *basePlugin) ensureFlock(flock *flock.Flock) error {
	err := RetryDuring(10*time.Second, 100*time.Millisecond, func() error {
		isLocked, err := flock.TryLock()
		if err != nil {
			return err
		}

		if !isLocked {
			return fmt.Errorf("could not obtain flock for [%s]", flock.Path())
		}

		return nil
	})

	return err
}

// NewPluginResource instantiates 'pluginResource' and links it with the given plugin.
func NewPluginResource(plugin Plugin) PluginResource {
	return &pluginResource{
		Resource: nagopher.NewResource(),
		plugin:   plugin,
	}
}

// Probe is an override for 'BaseResource.Probe(...)', which is being called by nagopher for collecting metrics. This
// method should never be overridden by any of the plugins, as it will just pass all arguments to 'Plugin.Probe()',
// where the plugins should define their actual check/metrics logic.
func (r *pluginResource) Probe(warnings nagopher.WarningCollection) ([]nagopher.Metric, error) {
	return r.plugin.Probe(warnings)
}

func (r *pluginResource) Plugin() Plugin {
	return r.plugin
}

// NewPluginSummarizer instantiates 'pluginSummarizer'.
func NewPluginSummarizer(plugin Plugin) PluginSummarizer {
	return &pluginSummarizer{
		Summarizer: nagopher.NewSummarizer(),
		plugin:     plugin,
	}
}

func (s pluginSummarizer) Plugin() Plugin {
	return s.plugin
}
