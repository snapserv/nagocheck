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

import (
	"fmt"
	"github.com/snapserv/nagopher"
	"io/ioutil"
	"strconv"
	"strings"
)

func (r *interfaceResource) Collect(warnings nagopher.WarningCollection) error {
	device := r.Plugin().InterfaceName

	if err := r.collectLinkState(device); err != nil {
		return err
	}

	if err := r.collectLinkSpeed(device); err != nil {
		warnings.Add(nagopher.NewWarning(err.Error()))
	}
	if err := r.collectLinkDuplex(device); err != nil {
		warnings.Add(nagopher.NewWarning(err.Error()))
	}
	if err := r.collectTransmitErrors(device); err != nil {
		warnings.Add(nagopher.NewWarning(err.Error()))
	}
	if err := r.collectReceiveErrors(device); err != nil {
		warnings.Add(nagopher.NewWarning(err.Error()))
	}

	return nil
}

func (r *interfaceResource) collectLinkState(device string) error {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/operstate", device))
	if err != nil {
		return fmt.Errorf("could not determine link state (%s)", err.Error())
	}

	r.linkState = strings.ToUpper(strings.TrimSpace(string(bytes)))
	return nil
}

func (r *interfaceResource) collectLinkSpeed(device string) error {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/speed", device))
	if err != nil {
		return fmt.Errorf("could not determine link speed (%s)", err.Error())
	}

	rawSpeed := strings.TrimSpace(string(bytes))
	speed, err := strconv.ParseInt(rawSpeed, 10, strconv.IntSize)
	if err != nil {
		return fmt.Errorf("could not parse link speed [%s] as integer (%s)", rawSpeed, err.Error())
	}

	r.linkSpeed = int(speed)
	return nil
}

func (r *interfaceResource) collectLinkDuplex(device string) error {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/duplex", device))
	if err != nil {
		return fmt.Errorf("could not determine link duplex (%s)", err.Error())
	}

	r.linkDuplex = strings.ToUpper(strings.TrimSpace(string(bytes)))
	return nil
}

func (r *interfaceResource) collectTransmitErrors(device string) error {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/tx_errors", device))
	if err != nil {
		return fmt.Errorf("could not determine transmit errors (%s)", err.Error())
	}

	rawErrorCount := strings.TrimSpace(string(bytes))
	errorCount, err := strconv.ParseInt(rawErrorCount, 10, strconv.IntSize)
	if err != nil {
		return fmt.Errorf("could not parse transmit errors [%s] as integer (%s)", rawErrorCount, err.Error())
	}

	r.transmitErrors = int(errorCount)
	return nil
}

func (r *interfaceResource) collectReceiveErrors(device string) error {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/rx_errors", device))
	if err != nil {
		return fmt.Errorf("could not determine receive errors (%s)", err.Error())
	}

	rawErrorCount := strings.TrimSpace(string(bytes))
	errorCount, err := strconv.ParseInt(rawErrorCount, 10, strconv.IntSize)
	if err != nil {
		return fmt.Errorf("could not parse receive errors [%s] as integer (%s)", rawErrorCount, err.Error())
	}

	r.receiveErrors = int(errorCount)
	return nil
}
