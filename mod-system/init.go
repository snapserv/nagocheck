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

package modsystem

import "github.com/snapserv/nagopher-checks/shared"

// GetModuleCommand returns a module command declaration for this check module
func GetModuleCommand() shared.ModuleCommand {
	return shared.ModuleCommand{
		Name:        "system",
		Description: "Operating System",
		PluginCommands: shared.PluginCommands{
			{
				Name:        "interface",
				Description: "Network Interface",
				Plugin:      newInterfacePlugin(),
			},
			{
				Name:        "load",
				Description: "Load Average",
				Plugin:      newLoadPlugin(),
			},
			{
				Name:        "memory",
				Description: "Memory Usage",
				Plugin:      newMemoryPlugin(),
			},
		},
	}
}
