package mounts // import "github.com/docker/docker/volume/mounts"

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

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

func lockedPathValidation(path string) error {
	call, err := middleware.Call("pool.dataset.path_in_locked_datasets", path)
	if err == nil {
		isLocked, ok := call["result"].(bool)
		if ok && isLocked {
			return errors.Errorf("Dataset path is locked")
		}
	}
	return nil
}

func isIXVolumePath(path string, datasetPath string) bool {
	releasePath := filepath.Join("mnt", datasetPath, "releases")
	if strings.Index(path, "/"+releasePath) == 0 {
		appPath := strings.Replace(path, "/"+releasePath+"/", "", 1)
		appName := strings.Split(appPath, "/")[0]
		volumePath := filepath.Join(releasePath, appName, "volumes", "ix_volumes")
		if strings.Contains(path, "/"+volumePath) {
			return true
		}
	}
	return false
}

func ignorePath(path string, dataset string) bool {
	ignorePaths := []string{"/etc/", "/sys/", "/proc/", "/var/lib/kubelet/pods/"}
	ignorePaths = append(ignorePaths, middleware.GetIgnorePaths()...)
	for _, igPath := range ignorePaths {
		if strings.Index(path, igPath) == 0 {
			return true
		}
	}
	if isIXVolumePath(path, dataset) {
		return true
	}
	return false
}

func getAttachments(path string) []string {
	attachments, err := middleware.Call("pool.dataset.attachments_with_path", path)
	if err == nil {
		attachmentsResults := attachments["result"].([]interface{})
		var attachmentList []string
		for _, attach := range attachmentsResults {
			attachNames := attach.(map[string]interface{})["attachments"].([]interface{})
			for _, name := range attachNames {
				attachmentList = append(attachmentList, name.(string))
			}
		}
		return attachmentList
	}
	return nil
}

func attachedPathValidation(path string) error {
	datasetPath := middleware.GetRootDataset()
	if ignorePath(path, datasetPath) {
		return nil
	}
	attachmentsResults := getAttachments(path)
	if attachmentsResults != nil && len(attachmentsResults) > 0 {
		attach := ""
		for _, pa := range attachmentsResults {
			attach += pa + ","
		}
		return errors.Errorf("Invalid mount path. %s. Following app uses this path. `%s`.", path, attach)
	}
	return nil
}

func pathToList(path string) []string {
	rawPathList := strings.Split(path, "/")
	var processPathList []string
	for _, name := range rawPathList {
		if name != "" {
			processPathList = append(processPathList, name)
		}
	}
	return processPathList
}

func ixMountValidation(path string) error {
	pathList := pathToList(path)
	blockPath := []string{"/cluster/ctdb_shared_vol", "/cluster"}
	if len(pathList) < 3 && pathList[0] == "mnt" {
		return errors.Errorf("Invalid path %s. Mounting root dataset or path outside a pool is not allowed", path)
	}
	for _, blPath := range blockPath {
		blPathLis := pathToList(blPath)
		if reflect.DeepEqual(pathList, blPathLis) {
			return errors.Errorf("Path %s is blocked and cannot be mounted.", path)
		}
	}
	return nil
}
