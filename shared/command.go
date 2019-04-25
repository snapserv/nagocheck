package shared

import "fmt"

// ModuleCommands represents a slice of ModuleCommand instances and offers a lookup method by name
type ModuleCommands []ModuleCommand

// PluginCommands represents a slice of PluginCommand instances and offers a lookup method by name
type PluginCommands []PluginCommand

// Command is a generic interface, which provides common methods for ModuleCommand and PluginCommand
type Command interface {
	Name() string
	Description() string
}

// ModuleCommand is an interface, which provides a set of plugin commands underneath a given module
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

// PluginCommand is a single plugin command of a module, which executes a specific check
type PluginCommand interface {
	Command
	Plugin() Plugin
}

type pluginCommand struct {
	name        string
	description string
	plugin      Plugin
}

// GetByName searches through a ModuleCommands slice and returns a module with the given name or an error, if not found
func (c ModuleCommands) GetByName(name string) (ModuleCommand, error) {
	for _, moduleCommand := range c {
		if moduleCommand.Name() == name {
			return moduleCommand, nil
		}
	}

	return nil, fmt.Errorf("could not find module command with name [%s]", name)
}

// GetByName searches through a PluginCommands slice and returns a module with the given name or an error, if not found
func (c PluginCommands) GetByName(name string) (PluginCommand, error) {
	for _, pluginCommand := range c {
		if pluginCommand.Name() == name {
			return pluginCommand, nil
		}
	}

	return nil, fmt.Errorf("could not find plugin command with name [%s]", name)
}

// NewModuleCommand instantiates a new ModuleCommand with the given options
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

// NewPluginCommand instantiates a new PluginCommand with the given options
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
