package shared

import "fmt"

type ModuleCommands []ModuleCommand
type PluginCommands []PluginCommand

type Command interface {
	Name() string
	Description() string
}

type ModuleCommand interface {
	Command
	Module() Module
	PluginCommands() PluginCommands
}

type moduleCommand struct {
	name           string
	description    string
	module         Module
	pluginCommands PluginCommands
}

type PluginCommand interface {
	Command
	Plugin() Plugin
}

type pluginCommand struct {
	name        string
	description string
	plugin      Plugin
}

func (c ModuleCommands) GetByName(name string) (ModuleCommand, error) {
	for _, moduleCommand := range c {
		if moduleCommand.Name() == name {
			return moduleCommand, nil
		}
	}

	return nil, fmt.Errorf("could not find module command with name [%s]", name)
}

func (c PluginCommands) GetByName(name string) (PluginCommand, error) {
	for _, pluginCommand := range c {
		if pluginCommand.Name() == name {
			return pluginCommand, nil
		}
	}

	return nil, fmt.Errorf("could not find plugin command with name [%s]", name)
}

func NewModuleCommand(name string, description string, module Module, pluginCommands ...PluginCommand) ModuleCommand {
	moduleCommand := &moduleCommand{
		name:           name,
		description:    description,
		module:         module,
		pluginCommands: pluginCommands,
	}

	return moduleCommand
}

func (c moduleCommand) Name() string {
	return c.name
}

func (c moduleCommand) Description() string {
	return c.description
}

func (c moduleCommand) Module() Module {
	return c.module
}

func (c moduleCommand) PluginCommands() PluginCommands {
	return c.pluginCommands
}

func NewPluginCommand(name string, description string, plugin Plugin) PluginCommand {
	pluginCommand := &pluginCommand{
		name:        name,
		description: description,
		plugin:      plugin,
	}

	return pluginCommand
}

func (c pluginCommand) Name() string {
	return c.name
}

func (c pluginCommand) Description() string {
	return c.description
}

func (c pluginCommand) Plugin() Plugin {
	return c.plugin
}
