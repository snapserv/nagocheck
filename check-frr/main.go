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

import "gopkg.in/alecthomas/kingpin.v2"

func main() {
	bgpNeighborCommand := kingpin.Command("bgp-neighbor", "FRRouting BGPd: Neighbor Check")
	bgpNeighborPlugin := newBgpNeighborPlugin()
	bgpNeighborPlugin.DefineFlags(bgpNeighborCommand)

	switch kingpin.Parse() {
	case "bgp-neighbor":
		bgpNeighborPlugin.Execute()
	}
}
