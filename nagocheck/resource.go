package nagocheck

import (
	"encoding/json"
	"fmt"
	"github.com/fabiokung/shm"
	"github.com/snapserv/nagopher"
	"io/ioutil"
	"os"
	"syscall"
)

type Resource interface {
	nagopher.Resource
	Plugin() Plugin
}

type ResourceOpt func(*baseResource)

type baseResource struct {
	nagopher.Resource
	plugin Plugin

	persistenceKey string
}

const shmOpenFlags = os.O_CREATE | os.O_RDONLY | syscall.O_DSYNC | syscall.O_RSYNC
const shmDefaultMode = 0600

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

func ResourcePersistence(uniqueKey string) ResourceOpt {
	return func(r *baseResource) {
		r.persistenceKey = r.Plugin().Name() + uniqueKey
	}
}

func (r baseResource) Probe(warnings nagopher.WarningCollection) ([]nagopher.Metric, error) {
	if err := r.loadPersistentData(); err != nil {
		return []nagopher.Metric{}, fmt.Errorf("nagopher: unable to load persistent data: %s", err.Error())
	}

	metrics, err := r.Resource.Probe(warnings)
	if err != nil {
		return metrics, err
	}

	if err := r.storePersistentData(); err != nil {
		return []nagopher.Metric{}, fmt.Errorf("nagopher: unable to store persistent data: %s", err.Error())
	}

	return metrics, err
}

func (r *baseResource) loadPersistentData() (rerr error) {
	// Skip persistence if identifier or store is missing
	if r.persistenceKey == "" {
		return nil
	}

	// Attempt to open or create file using SHM
	file, err := shm.Open(r.persistenceKey, shmOpenFlags, shmDefaultMode)
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
		if err := json.Unmarshal(jsonData, r); err != nil {
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
	jsonData, err := json.Marshal(r)
	if err != nil {
		return err
	}

	// Attempt to open or create file using SHM
	file, err := shm.Open(r.persistenceKey, shmOpenFlags, shmDefaultMode)
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
