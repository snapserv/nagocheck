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
	"fmt"
	"github.com/snapserv/nagocheck/nagocheck"
	"strings"
)

type frroutingModule struct {
	nagocheck.Module

	session Session

	connectionMode string
	vtyshCommand   string
}

// NewFrroutingModule instantiates frroutingModule and all contained plugins
func NewFrroutingModule() nagocheck.Module {
	return &frroutingModule{
		Module: nagocheck.NewModule("frrouting",
			nagocheck.ModuleDescription("FRRouting"),
			nagocheck.ModulePlugin(newBgpNeighborPlugin()),
		),
	}
}

func (m *frroutingModule) DefineFlags(node nagocheck.KingpinNode) {
	node.Flag("mode", "Specifies the connection mode for communicating with the FRRouting daemon.").
		Short('m').Default("vtysh").EnumVar(&m.connectionMode, "vtysh")

	node.Flag("vtysh-cmd", "[vtysh] Specifies the command with optional arguments to be used for executing vtysh. "+
		"Use comma to separate command and arguments. Example when using sudo: sudo,-n,/usr/bin/vtysh,-u").
		Default("/usr/bin/vtysh").StringVar(&m.vtyshCommand)
}

func (m *frroutingModule) ExecutePlugin(plugin nagocheck.Plugin) error {
	if m.connectionMode == "vtysh" {
		m.session = NewVtyshSession(strings.Split(m.vtyshCommand, ","))
	} else {
		return fmt.Errorf("unknown connection mode: " + m.connectionMode)
	}

	return m.Module.ExecutePlugin(plugin)
}
