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
	"github.com/snapserv/nagocheck/shared"
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
	moduleCommands := shared.ModuleCommands{
		modfrrouting.GetModuleCommand(),
		modsystem.GetModuleCommand(),
	}

	for _, moduleCommand := range moduleCommands {
		moduleDescription := "Check Module: " + moduleCommand.Description()
		moduleClause := kingpin.Command(moduleCommand.Name(), moduleDescription)
		moduleCommand.Module().DefineFlags(moduleClause)

		for _, pluginCommand := range moduleCommand.PluginCommands() {
			pluginDescription := fmt.Sprintf("%s: %s", moduleCommand.Description(), pluginCommand.Description())
			pluginClause := moduleClause.Command(pluginCommand.Name(), pluginDescription)
			pluginCommand.Plugin().DefineFlags(pluginClause)
		}
	}

	kingpin.Version(fmt.Sprintf("nagocheck, version %s (commit: %s)\nbuild date: %s, runtime: %s",
		BuildVersion, BuildCommit, BuildDate, runtime.Version()))
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.CommandLine.VersionFlag.Short('V')

	commandParts := strings.Split(kingpin.Parse(), " ")
	moduleCommand, err := moduleCommands.GetByName(commandParts[0])
	if err != nil {
		panic(err)
	}
	pluginCommand, err := moduleCommand.PluginCommands().GetByName(commandParts[1])
	if err != nil {
		panic(err)
	}

	moduleCommand.Module().Execute(pluginCommand.Plugin())
}
