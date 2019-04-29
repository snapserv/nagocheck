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

package nagocheck

import (
	"fmt"
	"github.com/snapserv/nagopher"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Module consists out of several plugins and offers methods for executing them
type Module interface {
	Name() string
	Description() string
	Plugins() map[string]Plugin

	DefineCommand() KingpinNode
	DefineFlags(node KingpinNode)
	RegisterPlugin(plugin Plugin)
	ExecutePlugin(plugin Plugin) error
	GetPluginByName(pluginName string) (Plugin, error)
}

// ModuleOpt is a type alias for functional options used by NewModule()
type ModuleOpt func(*baseModule)

type baseModule struct {
	name        string
	description string
	plugins     map[string]Plugin
}

// RegisterModules returns a map of modules with their name as the respective key. Additionally, all plugins contained
// by these modules are being registered to their respective module using Plugin.setModule()
func RegisterModules(modules ...Module) map[string]Module {
	result := make(map[string]Module)
	for _, module := range modules {
		result[module.Name()] = module
		for _, plugin := range module.Plugins() {
			plugin.setModule(module)
		}
	}

	return result
}

// NewModule instantiates baseModule with the given functional options
func NewModule(name string, options ...ModuleOpt) Module {
	module := &baseModule{
		name:        name,
		description: name,
		plugins:     make(map[string]Plugin),
	}

	for _, option := range options {
		option(module)
	}

	return module
}

// ModuleDescription is a functional option for NewModule(), which sets the module description
func ModuleDescription(description string) ModuleOpt {
	return func(m *baseModule) {
		m.description = description
	}
}

// ModulePlugin is a functional option for NewModule(), which registers a plugin using Module.RegisterPlugin()
func ModulePlugin(plugin Plugin) ModuleOpt {
	return func(m *baseModule) {
		m.RegisterPlugin(plugin)
	}
}

func (m *baseModule) RegisterPlugin(plugin Plugin) {
	m.plugins[plugin.Name()] = plugin
}

func (m *baseModule) DefineCommand() KingpinNode {
	moduleNode := kingpin.Command(m.name, m.description)

	for _, plugin := range m.plugins {
		pluginDescription := fmt.Sprintf("%s: %s", m.description, plugin.Description())
		pluginNode := moduleNode.Command(plugin.Name(), pluginDescription)

		plugin.defineDefaultFlags(pluginNode)
		plugin.DefineFlags(pluginNode)
	}

	return moduleNode
}

func (m *baseModule) DefineFlags(node KingpinNode) {
}

func (m *baseModule) ExecutePlugin(plugin Plugin) error {
	check := plugin.DefineCheck()
	runtime := nagopher.NewRuntime(plugin.VerboseOutput())
	runtime.ExecuteAndExit(check)

	return nil
}

func (m *baseModule) GetPluginByName(pluginName string) (Plugin, error) {
	plugin, ok := m.plugins[pluginName]
	if !ok {
		return nil, fmt.Errorf("plugin not found with name [%s]", pluginName)
	}

	return plugin, nil
}

func (m baseModule) Name() string {
	return m.name
}

func (m baseModule) Description() string {
	return m.description
}

func (m baseModule) Plugins() map[string]Plugin {
	return m.plugins
}
