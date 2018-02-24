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

import "fmt"

type ModuleCommands []ModuleCommand
type PluginCommands []PluginCommand

type ModuleCommand struct {
	Name           string
	Description    string
	PluginCommands PluginCommands
}

type PluginCommand struct {
	Name        string
	Description string
	Plugin      Plugin
}

func (mc ModuleCommands) GetByName(name string) (command ModuleCommand, _ error) {
	for _, moduleCommand := range mc {
		if moduleCommand.Name == name {
			return moduleCommand, nil
		}
	}

	return command, fmt.Errorf("could not find module command: %s", name)
}

func (pc PluginCommands) GetByName(name string) (command PluginCommand, _ error) {
	for _, pluginCommand := range pc {
		if pluginCommand.Name == name {
			return pluginCommand, nil
		}
	}

	return command, fmt.Errorf("could not find plugin command: %s", name)
}
