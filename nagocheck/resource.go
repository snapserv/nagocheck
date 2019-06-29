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

package nagocheck

import (
	"encoding/json"
	"fmt"
	"github.com/fabiokung/shm"
	"github.com/snapserv/nagopher"
	"io/ioutil"
	"strings"
)

// Resource provides a base type for nagocheck resources, which embeds nagopher.Resource
type Resource interface {
	nagopher.Resource
	Plugin() Plugin
}

// ResourceOpt is a type alias for functional options used by NewSummarizer()
type ResourceOpt func(*baseResource)

type baseResource struct {
	nagopher.Resource `json:"-"`
	plugin            Plugin

	persistenceKey   string
	persistenceStore interface{}
}

// NewResource instantiates baseResource with the given functional options
func NewResource(plugin Plugin, options ...ResourceOpt) Resource {
	resource := &baseResource{
		Resource: nagopher.NewResource(),
		plugin:   plugin,
	}

	for _, option := range options {
		option(resource)
	}

	return resource
}

// ResourcePersistence is a functional option for NewResource(), which enables resource persistence with the given key
func ResourcePersistence(uniqueKey string, dataStore interface{}) ResourceOpt {
	return func(r *baseResource) {
		r.persistenceKey = strings.ToLower(".nagocheck-" + r.Plugin().Name() + "-" + uniqueKey)
		r.persistenceStore = dataStore
	}
}

func (r baseResource) Setup(warnings nagopher.WarningCollection) error {
	if err := r.loadPersistentData(); err != nil {
		return fmt.Errorf("unable to load persistent data: %s", err.Error())
	}

	return nil
}

func (r baseResource) Teardown(warnings nagopher.WarningCollection) error {
	if err := r.storePersistentData(); err != nil {
		return fmt.Errorf("unable to store persistent data: %s", err.Error())
	}

	return nil
}

func (r *baseResource) loadPersistentData() (rerr error) {
	// Skip persistence if identifier or store is missing
	if r.persistenceKey == "" {
		return nil
	}

	// Attempt to open or create file using SHM
	file, err := shm.Open(r.persistenceKey, shmReadFlags, shmDefaultMode)
	if err != nil {
		return err
	}

	// Ensure file is always being properly closed
	defer func() {
		err := file.Close()
		if err != nil {
			rerr = err
		}
	}()

	// Attempt to read contents from file
	jsonData, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	// Attempt to unmarshal contents as JSON into target
	if len(jsonData) > 0 {
		if err := json.Unmarshal(jsonData, r.persistenceStore); err != nil {
			return err
		}
	}

	return nil
}

func (r baseResource) storePersistentData() (rerr error) {
	// Skip persistence if identifier or store is missing
	if r.persistenceKey == "" {
		return nil
	}

	// Attempt to marshal source into JSON
	jsonData, err := json.Marshal(r.persistenceStore)
	if err != nil {
		return err
	}

	// Attempt to open or create file using SHM
	file, err := shm.Open(r.persistenceKey, shmWriteFlags, shmDefaultMode)
	if err != nil {
		return err
	}

	// Ensure file is always being properly closed
	defer func() {
		err := file.Close()
		if err != nil {
			rerr = err
		}
	}()

	// Attempt to write JSON data into file
	if _, err := file.Write(jsonData); err != nil {
		return err
	}

	return nil
}

func (r *baseResource) Plugin() Plugin {
	return r.plugin
}
