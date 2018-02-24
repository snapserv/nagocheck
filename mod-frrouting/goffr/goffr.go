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

package goffr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/goexpect"
	"github.com/ziutek/telnet"
	"strings"
	"time"
)

// Session represents a goffr session to one or more instances of the FRRouting daemon
type Session struct {
	hostname  string
	password  string
	instances map[string]*Instance
}

// Instance represents a goffr instance, which is a lightweight wrapper around goexpect and a telnet session to the
// according daemon of FRRouting. This structure should never be instantiated directly and gets created by the
// Session.GetInstance() method.
type Instance struct {
	name           string
	systemName     string
	systemAddress  string
	systemPassword string
	expecter       expect.Expecter
}

// These constants represent an 'Enum' for all available FRRouting daemon names.
const (
	InstanceZebra = "zebra"
	InstanceRIP   = "ripd"
	InstanceRIPng = "ripngd"
	InstanceBGP   = "bgpd"
	InstanceOSPF  = "ospfd"
	InstanceOSPF6 = "ospf6d"
	InstanceISIS  = "isisd"
	InstancePIM   = "pimd"
)

const timeout = 5 * time.Second

// NewSession instantiates a new 'Session' without any instances.
func NewSession(hostname string, password string) *Session {
	return &Session{
		hostname:  hostname,
		password:  password,
		instances: make(map[string]*Instance),
	}
}

// GetInstance returns the instance with the given (daemon) name if already requested in a previous call. If no such
// instance was instantiated so far, a new instance gets automatically created, which tries connecting to the target
// daemon.
func (s *Session) GetInstance(name string) (*Instance, error) {
	instancePorts := map[string]int{
		"zebra":  2601,
		"ripd":   2602,
		"ripngd": 2603,
		"ospfd":  2604,
		"bgpd":   2605,
		"ospf6d": 2606,
		"isisd":  2607,
		"pimd":   2611,
	}

	if instance, ok := s.instances[name]; ok {
		return instance, nil
	}

	name = strings.TrimSpace(strings.ToLower(name))
	if instancePort, ok := instancePorts[name]; ok {
		instance := newInstance(name, fmt.Sprintf("%s:%d", s.hostname, instancePort), s.password)
		if err := instance.initialize(); err != nil {
			return nil, fmt.Errorf("goffr: could not initialize instance (%s)", err.Error())
		}

		s.instances[name] = instance
		return instance, nil
	}

	return nil, fmt.Errorf("goffr: unknown instance name [%s]", name)
}

func newInstance(name string, systemAddress string, systemPassword string) *Instance {
	return &Instance{
		name:           name,
		systemAddress:  systemAddress,
		systemPassword: systemPassword,
		expecter:       nil,
	}
}

// Execute tries to execute a command against the FRRouting daemon for which this instance was created. Please note that
// errors spewed out by the FRRouting daemon are not being handled, only connection/transmission errors. It is the
// callers duty to manually parse the output according to the FRRouting specifications.
func (i *Instance) Execute(command string) (string, error) {
	if err := i.prepare(); err != nil {
		return "", fmt.Errorf("gofrr-%s: could not prepare execution of command (%s)", i.name, err.Error())
	}

	result, err := i.expecter.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: command + "\n"},
		&expect.BExp{R: command + `\r\n([\s\S]+)` + i.promptLine()},
	}, timeout)
	if err != nil {
		return "", fmt.Errorf("gofrr-%s: command execution failed (%s)", i.name, err.Error())
	}

	output := strings.TrimSpace(result[0].Match[1])
	return output, nil
}

// ExecuteJSON is a lightweight wrapper against 'Execute()', which will try to parse and compact the output of the given
// command as JSON. In case the output does not represent valid JSON (e.g. an error occurred during the execution of the
// command), an error will be returned instead.
func (i *Instance) ExecuteJSON(command string) (string, error) {
	rawOutput, err := i.Execute(command)
	if err != nil {
		return "", err
	}

	compactOutput := new(bytes.Buffer)
	err = json.Compact(compactOutput, []byte(rawOutput))
	if err != nil {
		return "", fmt.Errorf("gofrr-%s: could not parse output [%s] as JSON (%s)",
			i.name, rawOutput, err.Error())
	}

	return compactOutput.String(), nil
}

func (i *Instance) initialize() error {
	var err error

	if i.expecter != nil {
		return fmt.Errorf("goffr-%s: already connected to system [%s]", i.name, i.systemAddress)
	}

	i.expecter, _, err = spawnTelnetExpecter(i.systemAddress, timeout)
	if err != nil {
		return fmt.Errorf("goffr-%s: could not connect to system [%s] (%s)", i.name, i.systemAddress, err.Error())
	}

	result, err := i.expecter.ExpectBatch([]expect.Batcher{
		&expect.BExp{R: `[\s\S]+\r\nUser Access Verification\r\n\r\nPassword: `},
		&expect.BSnd{S: i.systemPassword + "\n"},
		&expect.BExp{R: `\r\n([^>]+)>\s+$`},
	}, timeout)
	if err != nil {
		return fmt.Errorf("gofrr-%s: could not authenticate against system (%s)", i.name, err.Error())
	}

	i.systemName = strings.TrimSpace(result[1].Match[1])
	_, err = i.expecter.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: "terminal length 0\n"},
		&expect.BExp{R: i.promptLine()},
	}, timeout)
	if err != nil {
		return fmt.Errorf("gofrr-%s: could not disable terminal paging (%s)", i.name, err.Error())
	}

	return nil
}

func (i *Instance) prepare() error {
	_, err := i.expecter.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: i.promptLine()},
	}, timeout)

	return err
}

func (i *Instance) promptLine() string {
	return `\r\n` + i.systemName + `> $`
}

func spawnTelnetExpecter(address string, timeout time.Duration, opts ...expect.Option) (expect.Expecter, <-chan error, error) {
	connection, err := telnet.Dial("tcp", address)
	if err != nil {
		return nil, nil, err
	}

	resultChannel := make(chan error)
	return expect.SpawnGeneric(&expect.GenOptions{
		In:  connection,
		Out: connection,
		Wait: func() error {
			return <-resultChannel
		},
		Close: func() error {
			close(resultChannel)
			return connection.Close()
		},
		Check: func() bool {
			return true
		},
	}, timeout, opts...)
}
