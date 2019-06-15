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
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const timeout = 10 * time.Second

// Session represents an active connection for communicating with FRRouting
type Session interface {
	GetBgpNeighbors() ([]*BgpNeighbor, error)
	GetBgpNeighbor(neighborAddress string) (*BgpNeighbor, error)
}

type vtyshSession struct {
	vtyshCommand []string
}

// BgpNeighbor contains config and operational data about a BGP neighbor/peer
type BgpNeighbor struct {
	LocalRouterID string `json:"localRouterId"`
	LocalHost     string `json:"localHost"`
	LocalPort     uint16 `json:"localPort"`
	LocalAS       uint32 `json:"localAs"`

	RemoteRouterID string `json:"remoteRouterId"`
	RemoteHost     string `json:"remoteHost"`
	RemotePort     uint16 `json:"remotePort"`
	RemoteAS       uint32 `json:"remoteAs"`

	Version            uint8  `json:"bgpVersion"`
	OperationalState   string `json:"bgpState"`
	Description        string `json:"nbrDesc"`
	UpTimer            uint64 `json:"bgpTimerUpMsec"`
	ResetTimer         uint64 `json:"lastResetTimerMsecs"`
	ResetReason        string `json:"lastResetDueTo"`
	NotificationReason string `json:"lastNotificationReason"`
	UpdateSource       string `json:"updateSource"`

	AddressFamilies map[string]BgpNeighborAddressFamily `json:"addressFamilyInfo"`

	LastStateChange  time.Duration
	PrefixUsageTotal uint64
	PrefixLimitTotal uint64
}

// BgpNeighborAddressFamily contains config and operational data about a specific address family of a neighbor/peer
type BgpNeighborAddressFamily struct {
	PeerGroup   string `json:"peerGroupMember"`
	PrefixCount uint64 `json:"acceptedPrefixCounter"`
	PrefixLimit uint64 `json:"prefixAllowedMax"`
}

// NewVtyshSession instantiates a new Session which will use vtysh to communicate with FRRouting
func NewVtyshSession(vtyshCommand []string) Session {
	return &vtyshSession{
		vtyshCommand: vtyshCommand,
	}
}

func (s *vtyshSession) GetBgpNeighbors() ([]*BgpNeighbor, error) {
	jsonData, err := s.executeJSON("show bgp neighbor json")
	if err != nil {
		return nil, fmt.Errorf("could not fetch neighborsMap data: %s", err.Error())
	}

	neighborsMap, err := s.parseBgpNeighbors([]byte(jsonData))
	if err != nil {
		return nil, fmt.Errorf("could not parse neighborsMap data: %s", err.Error())
	}

	neighbors := make([]*BgpNeighbor, 0, len(neighborsMap))
	for _, value := range neighborsMap {
		neighbors = append(neighbors, value)
	}

	return neighbors, nil
}

func (s *vtyshSession) GetBgpNeighbor(neighborAddress string) (*BgpNeighbor, error) {
	jsonData, err := s.executeJSON("show bgp neighbor %s json", neighborAddress)
	if err != nil {
		return nil, fmt.Errorf("could not fetch neighbor data: %s", err.Error())
	}

	neighbors, err := s.parseBgpNeighbors([]byte(jsonData))
	if err != nil {
		return nil, fmt.Errorf("could not parse neighbor data: %s", err.Error())
	}

	neighbor, ok := neighbors[neighborAddress]
	if !ok {
		return nil, fmt.Errorf("could not find neighbor [%s]", neighborAddress)
	}

	return neighbor, nil
}

func (s *vtyshSession) parseBgpNeighbors(jsonData []byte) (map[string]*BgpNeighbor, error) {
	neighbors := make(map[string]*BgpNeighbor)
	if err := json.Unmarshal(jsonData, &neighbors); err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON neighbor data: %s", err.Error())
	}

	for neighborAddress, neighbor := range neighbors {
		neighbor.OperationalState = strings.ToUpper(neighbor.OperationalState)
		if neighbor.LocalHost == "" {
			neighbor.LocalHost = neighbor.UpdateSource
		}
		if neighbor.RemoteHost == "" {
			neighbor.RemoteHost = neighborAddress
		}

		if neighbor.UpTimer > 0 {
			neighbor.LastStateChange = time.Duration(neighbor.UpTimer) * time.Millisecond
		} else {
			neighbor.LastStateChange = time.Duration(neighbor.ResetTimer) * time.Millisecond
		}

		for _, addressFamily := range neighbor.AddressFamilies {
			neighbor.PrefixUsageTotal += addressFamily.PrefixCount
			neighbor.PrefixLimitTotal += addressFamily.PrefixLimit
		}
	}

	return neighbors, nil
}

func (s *vtyshSession) execute(commandFmt string, args ...interface{}) (_ string, err error) {
	cmdArgs := append(s.vtyshCommand, "-c", fmt.Sprintf(commandFmt, args...))
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)

	timer := time.AfterFunc(timeout, func() {
		err = fmt.Errorf("command execution timed out after %f seconds", timeout.Seconds())
		_ = cmd.Process.Kill()
	})
	output, err := cmd.CombinedOutput()
	timer.Stop()

	return string(output), err
}

func (s *vtyshSession) executeJSON(commandFmt string, args ...interface{}) (_ string, err error) {
	rawOutput, err := s.execute(commandFmt, args...)
	sanitizedOutput := strings.Replace(strings.TrimSpace(rawOutput), "\n", " ", -1)
	if err != nil {
		return "", fmt.Errorf("command execution failed: %s (%s)", err.Error(), sanitizedOutput)
	}

	jsonBuffer := new(bytes.Buffer)
	err = json.Compact(jsonBuffer, []byte(rawOutput))
	if err != nil {
		return "", fmt.Errorf("could not parse output [%s] as JSON: %s", sanitizedOutput, err.Error())
	}

	return jsonBuffer.String(), nil
}
