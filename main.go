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

package main

import (
	"fmt"
	"github.com/snapserv/nagopher-checks/mod-frrouting"
	"github.com/snapserv/nagopher-checks/mod-system"
	"github.com/snapserv/nagopher-checks/shared"
	"gopkg.in/alecthomas/kingpin.v2"
	"strings"
)

func main() {
	moduleCommands := shared.ModuleCommands{
		modsystem.GetModuleCommand(),
		modfrrouting.GetModuleCommand(),
	}

	for _, moduleCommand := range moduleCommands {
		moduleDescription := "Check Module: " + moduleCommand.Description
		moduleClause := kingpin.Command(moduleCommand.Name, moduleDescription)
		for _, pluginCommand := range moduleCommand.PluginCommands {
			pluginDescription := fmt.Sprintf("%s: %s", moduleCommand.Description, pluginCommand.Description)
			pluginClause := moduleClause.Command(pluginCommand.Name, pluginDescription)
			pluginCommand.Plugin.DefineFlags(pluginClause)
		}
	}

	commandParts := strings.Split(kingpin.Parse(), " ")
	moduleCommand, err := moduleCommands.GetByName(commandParts[0])
	if err != nil {
		panic(err)
	}
	pluginCommand, err := moduleCommand.PluginCommands.GetByName(commandParts[1])
	if err != nil {
		panic(err)
	}

	pluginCommand.Plugin.Execute()
}
