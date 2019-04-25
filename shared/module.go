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

// Module collects several plugin commands underneath a module command and offers the possibility to define CLI flags
type Module interface {
	DefineFlags(KingpinNode)
	Execute(Plugin)
	GetModuleCommand() ModuleCommand
}

type baseModule struct{}

// NewBaseModule instantiates a new BaseModule, which should be inherited by user-defined module types
func NewBaseModule() Module {
	return &baseModule{}
}

func (m *baseModule) DefineFlags(kp KingpinNode) {}

func (m *baseModule) Execute(plugin Plugin) {
	plugin.Execute()
}

func (m *baseModule) GetModuleCommand() ModuleCommand {
	return nil
}
