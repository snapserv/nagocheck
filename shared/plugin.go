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
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/snapserv/nagopher"
	"github.com/theckman/go-flock"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Plugin represents a interface for all plugin types.
type Plugin interface {
	ParseFlags()
	Probe(*nagopher.WarningCollection) ([]nagopher.Metric, error)
}

// BasePlugin represents a generic plugin from which all other plugin types should originate.
type BasePlugin struct {
	Verbose       bool
	WarningRange  *nagopher.Range
	CriticalRange *nagopher.Range
}

// BasePluginResource presents a generic nagopher 'BaseResource' linked to a plugin.
type BasePluginResource struct {
	*nagopher.BaseResource
	basePlugin Plugin
}

// NewPlugin instantiates 'BasePlugin'.
func NewPlugin() *BasePlugin {
	return &BasePlugin{}
}

// ParseFlags adds several flag definitions for parsing followed by executing 'flag.Parse()'. Afterwards, the threshold
// ranges (if given) are being parsed.
func (p *BasePlugin) ParseFlags() {
	var err error
	var warningRangeString, criticalRangeString string

	kingpin.Flag("verbose", "Enable verbose plugin output.").
		Short('v').BoolVar(&p.Verbose)
	kingpin.Flag("warning", "Warning threshold formatted as Nagios range specifier.").
		Short('w').StringVar(&warningRangeString)
	kingpin.Flag("critical", "Critical threshold formatted as Nagios range specifier.").
		Short('c').StringVar(&criticalRangeString)
	kingpin.Parse()

	if p.WarningRange, err = nagopher.ParseRange(warningRangeString); err != nil {
		panic(err.Error())
	}
	if p.CriticalRange, err = nagopher.ParseRange(criticalRangeString); err != nil {
		panic(err.Error())
	}
}

// Execute is a helper method which creates a new nagopher 'Runtime', executes a check and exits
func (p *BasePlugin) Execute(check *nagopher.Check) {
	runtime := nagopher.NewRuntime(p.Verbose)
	runtime.ExecuteAndExit(check)
}

// ExecutePersistent is a helper method which extends Execute() with flock (based on given unique key, which should be
// chosen wisely) and a persistent store, which is also named by the unique key passed. This is especially useful when
// used with contexts like 'DeltaContext', which compare the current measurement against a previously measurement.
func (p *BasePlugin) ExecutePersistent(check *nagopher.Check, uniqueKey string, store interface{}) {
	// Prefix unique key with 'nagopher-checks.'
	uniqueKey = "nagopher-checks." + uniqueKey

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
	runtime := nagopher.NewRuntime(p.Verbose)
	result := runtime.Execute(check)

	// Save plugin persistence store
	if err := SavePersistentStore(uniqueKey, store); err != nil {
		panic(err)
	}

	// Unlink and unlock flock immediately after execution
	syscall.Unlink(fileLock.Path())
	fileLock.Unlock()

	// Print plugin output and exit with the according exit code
	fmt.Print(result.Output)
	os.Exit(result.ExitCode)
}

// Probe represents the method executing the actual check/metrics logic and should be overridden by each plugin for
// returning metrics. It also supports adding warnings through the passed 'WarningCollection' or returning an error in
// case metric collection goes wrong.
func (p *BasePlugin) Probe(warnings *nagopher.WarningCollection) (metrics []nagopher.Metric, _ error) {
	return metrics, nil
}

func (p *BasePlugin) createFlock(identifier string) *flock.Flock {
	return flock.NewFlock(fmt.Sprintf("/tmp/.%s.lock", identifier))
}

func (p *BasePlugin) ensureFlock(flock *flock.Flock) error {
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

// NewPluginResource instantiates 'BasePluginResource' and links it with the given plugin.
func NewPluginResource(plugin Plugin) *BasePluginResource {
	return &BasePluginResource{
		BaseResource: nagopher.NewResource(),
		basePlugin:   plugin,
	}
}

// Probe is an override for 'BaseResource.Probe(...)', which is being called by nagopher for collecting metrics. This
// method should never be overridden by any of the plugins, as it will just pass all arguments to 'Plugin.Probe()',
// where the plugins should define their actual check/metrics logic.
func (pr *BasePluginResource) Probe(warnings *nagopher.WarningCollection) ([]nagopher.Metric, error) {
	return pr.basePlugin.Probe(warnings)
}
