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
	"bufio"
	"errors"
	"fmt"
	"github.com/snapserv/nagopher"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const zfsProcBasePath = "/proc/spl/kstat/zfs"
const zfsPoolPathPattern = "/*/io"

const (
	zfsTypeUint64 = "4"
)

func (r *zfsResource) Collect(warnings nagopher.WarningCollection) error {
	if err := r.collectPools(zfsProcBasePath); err != nil {
		return err
	}

	return nil
}

func (r *zfsResource) collectPools(basePath string) error {
	globMatches, err := filepath.Glob(filepath.Join(zfsProcBasePath, zfsPoolPathPattern))
	if err != nil {
		return fmt.Errorf("could not glob zfs pool paths: %s", err.Error())
	}
	if globMatches == nil {
		return nil
	}

	r.poolStats = make(map[string]zfsPoolStats)
	for _, globMatch := range globMatches {
		poolPath := filepath.Dir(globMatch)
		poolName := filepath.Base(poolPath)
		poolStats, err := r.updatePoolStats(poolPath)

		if err != nil {
			return fmt.Errorf("could not gather zfs pool statistics: %s", err.Error())
		}

		r.poolStats[poolName] = poolStats
	}

	return nil
}

func (r *zfsResource) updatePoolStats(poolPath string) (stats zfsPoolStats, _ error) {
	stateFile, err := os.Open(filepath.Join(poolPath, "state"))
	if err != nil {
		return stats, fmt.Errorf("could not open state file: %s", err.Error())
	}
	defer func() {
		_ = stateFile.Close()
	}()

	ioStatsFile, err := os.Open(filepath.Join(poolPath, "io"))
	if err != nil {
		return stats, fmt.Errorf("could not open i/o stats file: %s", err.Error())
	}
	defer func() {
		_ = ioStatsFile.Close()
	}()

	stats.state, err = r.parsePoolState(stateFile)
	if err != nil {
		return stats, fmt.Errorf("could not gather state: %s", err.Error())
	}

	stats.io, err = r.parsePoolIOStats(ioStatsFile)
	if err != nil {
		return stats, fmt.Errorf("could not gather i/o stats: %s", err.Error())
	}

	return stats, nil
}

func (r *zfsResource) parsePoolState(reader io.Reader) (string, error) {
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		err := scanner.Err()
		if err == nil {
			return "", errors.New("could not read state: EOF")
		}

		return "", fmt.Errorf("could not read state: %s", scanner.Err())
	}

	state := strings.ToUpper(strings.TrimSpace(scanner.Text()))

	return state, nil
}

func (r *zfsResource) parsePoolIOStats(reader io.Reader) (stats zfsPoolIOStats, _ error) {
	var fields []string

	skipParsing := true
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())

		if skipParsing && len(parts) >= 12 && parts[0] == "nread" {
			skipParsing = false
			fields = make([]string, len(parts))
			copy(fields, parts)
			continue
		} else if skipParsing {
			continue
		}

		for index, key := range fields {
			value, err := strconv.ParseUint(parts[index], 10, 64)
			if err != nil {
				return stats, fmt.Errorf("could not parse unsigned integer for %s: %s", key, err.Error())
			}

			switch key {
			case "reads":
				stats.readCount = value
			case "writes":
				stats.writeCount = value
			case "nread":
				stats.bytesRead = value
			case "nwritten":
				stats.bytesWritten = value
			}
		}
	}

	return stats, nil
}
