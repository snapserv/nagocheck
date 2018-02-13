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

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

func getInterfaceStats(name string) (*interfaceStats, error) {
	state, err := getInterfaceState(name)
	if err != nil {
		return nil, err
	}

	speed, err := getInterfaceSpeed(name)
	if err != nil {
		return nil, err
	}

	duplex, err := getInterfaceDuplex(name)
	if err != nil {
		return nil, err
	}

	return &interfaceStats{
		State:  state,
		Speed:  speed,
		Duplex: duplex,
	}, nil
}

func getInterfaceState(device string) (string, error) {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/operstate", device))
	if err != nil {
		return "", fmt.Errorf(
			"interface: could not read /sys/class/net/<interface>/operstate file (%s)", err.Error())
	}

	return strings.ToUpper(strings.TrimSpace(string(bytes))), nil
}

func getInterfaceSpeed(device string) (int, error) {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/speed", device))
	if err != nil {
		return -1, fmt.Errorf(
			"interface: could not read /sys/class/net/<interface>/duplex file (%s)", err.Error())
	}

	rawSpeed := strings.TrimSpace(string(bytes))
	speed, err := strconv.ParseInt(rawSpeed, 10, strconv.IntSize)
	if err != nil {
		return -1, fmt.Errorf("interface: could not parse interface speed [%s] as integer (%s)",
			rawSpeed, err.Error())
	}

	return int(speed), nil
}

func getInterfaceDuplex(device string) (string, error) {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/duplex", device))
	if err != nil {
		return "", fmt.Errorf(
			"interface: could not read /sys/class/net/<interface>/duplex file (%s)", err.Error())
	}

	return strings.ToUpper(strings.TrimSpace(string(bytes))), nil
}
