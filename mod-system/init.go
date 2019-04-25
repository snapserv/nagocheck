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

package modsystem

import "github.com/snapserv/nagocheck/shared"

type systemModule struct {
	shared.Module
}

// GetModuleCommand is a helper method for instantiating 'systemModule' and calling the 'GetModuleCommand' method to
// return a module command declaration.
func GetModuleCommand() shared.ModuleCommand {
	return newSystemModule().GetModuleCommand()
}

func newSystemModule() shared.Module {
	return &systemModule{
		Module: shared.NewBaseModule(),
	}
}

func (m *systemModule) GetModuleCommand() shared.ModuleCommand {
	return shared.NewModuleCommand(
		"system", "Operating System", m,
		shared.NewPluginCommand("interface", "Network Interface", newInterfacePlugin()),
		shared.NewPluginCommand("load", "Load Average", newLoadPlugin()),
		shared.NewPluginCommand("memory", "Memory Usage", newMemoryPlugin()),
	)
}
