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

package modfrrouting

import (
	"github.com/snapserv/nagocheck/mod-frrouting/goffr"
	"github.com/snapserv/nagocheck/shared"
)

type frroutingModule struct {
	*shared.BaseModule

	GoffrSession goffr.Session

	ConnectionMode string
	VtyshPath      string
	TelnetAddress  string
	TelnetPassword string
}

// GetModuleCommand is a helper method for instantiating 'frroutingModule' and calling the 'GetModuleCommand' method to
// return a module command declaration.
func GetModuleCommand() shared.ModuleCommand {
	return newFrroutingModule().GetModuleCommand()
}

func newFrroutingModule() *frroutingModule {
	return &frroutingModule{}
}

func (m *frroutingModule) DefineFlags(kp shared.KingpinInterface) {
	kp.Flag("mode", "Specifies the mode which should be used to connect to the FRRouting daemon, which can either be "+
		"vtysh (recommended) or telnet.").
		Short('m').
		Default("vtysh").
		EnumVar(&m.ConnectionMode, "vtysh", "telnet")

	kp.Flag("vtysh-path", "Vtysh Mode: Absolute path to executable vtysh binary.").
		Default("/usr/bin/vtysh").
		StringVar(&m.VtyshPath)

	kp.Flag("telnet-address", "Telnet Mode: Specifies the address of the given router, which should offer a telnet "+
		"connection to the standard port used by FRRouting for the bgp daemon.").
		Default("localhost").
		StringVar(&m.TelnetAddress)

	kp.Flag("telnet-password", "Telnet Mode: Specifies the password which should be used for connecting against the "+
		"FRRouting bgpd daemon. Please note that this is the connection and -not- the enable password.").
		Default("example").
		StringVar(&m.TelnetPassword)
}

func (m *frroutingModule) Execute(plugin shared.Plugin) {
	if m.ConnectionMode == "vtysh" {
		m.GoffrSession = goffr.NewVtyshSession(m.VtyshPath)
	} else if m.ConnectionMode == "telnet" {
		m.GoffrSession = goffr.NewTelnetSession(m.TelnetAddress, m.TelnetPassword)
	} else {
		panic("unknown connection mode: " + m.ConnectionMode)
	}

	plugin.Execute()
}

func (m *frroutingModule) GetModuleCommand() shared.ModuleCommand {
	return shared.ModuleCommand{
		Name:        "frrouting",
		Description: "FRRouting",
		Module:      m,
		PluginCommands: shared.PluginCommands{
			{
				Name:        "bgp-neighbor",
				Description: "BGP Neighbor",
				Plugin:      newBgpNeighborPlugin(m),
			},
		},
	}
}
