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

package main

import (
	"fmt"
	"github.com/snapserv/nagocheck/mod-frrouting"
	"github.com/snapserv/nagocheck/mod-system"
	"github.com/snapserv/nagocheck/nagocheck"
	"gopkg.in/alecthomas/kingpin.v2"
	"runtime"
	"strings"
)

// Build variables, automatically set during compilation
var (
	BuildVersion = "SNAPSHOT"
	BuildCommit  = "N/A"
	BuildDate    = "N/A"
)

func main() {
	modules := nagocheck.RegisterModules(
		modfrrouting.NewFrroutingModule(),
		modsystem.NewSystemModule(),
	)

	kingpin.Version(fmt.Sprintf("nagocheck, version %s (commit: %s)\nbuild date: %s, runtime: %s",
		BuildVersion, BuildCommit, BuildDate, runtime.Version()))
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.CommandLine.VersionFlag.Short('V')

	for _, module := range modules {
		moduleNode := module.DefineCommand()
		module.DefineFlags(moduleNode)
	}

	commandParts := strings.Split(kingpin.Parse(), " ")
	module, ok := modules[commandParts[0]]
	if !ok {
		panic(fmt.Sprintf("module not found with name [%s]", commandParts[0]))
	}

	plugin, err := module.GetPluginByName(commandParts[1])
	if err != nil {
		panic(fmt.Sprintf("plugin not found with name [%s]", commandParts[1]))
	}

	if err := module.ExecutePlugin(plugin); err != nil {
		panic(fmt.Sprintf("plugin execution of [%s] failed: %s", commandParts[1], err.Error()))
	}
}
