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

package shared

import (
	"encoding/json"
	"github.com/fabiokung/shm"
	"io/ioutil"
	"os"
	"syscall"
)

// LoadPersistentStore loads a persistent shm-based store, which can represent any structure/type as long as it can be
// unmarshalled by the builtin 'json.Unmarshal' function. Please note that this function should only be called when
// protected by a flock, as several processes operating the same store can lead to data loss.
func LoadPersistentStore(identifier string, v interface{}) error {
	file, err := shm.Open(identifier, os.O_CREATE|os.O_RDONLY|syscall.O_DSYNC|syscall.O_RSYNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	if len(bytes) > 0 {
		if err = json.Unmarshal(bytes, v); err != nil {
			return err
		}
	}

	return nil
}

// SavePersistentStore stores a persistent shm-based store, which can represent any structure/type as long as it can be
// marshalled by the builtin 'json.Marshal' function. Please note that this function should only be called when
// protected by a flock, as several processes operating the same store can lead to data loss.
func SavePersistentStore(identifier string, v interface{}) error {
	bytes, err := json.Marshal(v)
	if err != nil {
		return err
	}

	file, err := shm.Open(identifier, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|syscall.O_DSYNC|syscall.O_RSYNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err = file.Write(bytes); err != nil {
		return err
	}

	return nil
}
