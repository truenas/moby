package mounts // import "github.com/docker/docker/volume/mounts"

import (
	"fmt"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/middleware"
	"github.com/pkg/errors"
)

type errMountConfig struct {
	mount *mount.Mount
	err   error
}

func (e *errMountConfig) Error() string {
	return fmt.Sprintf("invalid mount config for type %q: %v", e.mount.Type, e.err.Error())
}

func errBindSourceDoesNotExist(path string) error {
	return errors.Errorf("bind source path does not exist: %s", path)
}

func errExtraField(name string) error {
	return errors.Errorf("field %s must not be specified", name)
}
func errMissingField(name string) error {
	return errors.Errorf("field %s must not be empty", name)
}

func hostPathValidation(path string) error {
	validationErr, err := middleware.Call("chart.release.validate_host_source_path", path)
	if err == nil && validationErr != nil {
		return errors.Errorf(validationErr.(string))
	}
	return nil
}
