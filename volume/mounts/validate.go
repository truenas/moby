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
		isLocked, ok := call.(bool)
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

func ignorePath(path string) bool {
	// "/" and "/home/keys/" are added for openebs use only, regular containers can't mount "/" as we have validation
	// already in place by docker elsewhere to prevent that from happening
	if path == "/" {
		return true
	}
	ignorePaths := []string{"/etc/", "/sys/", "/proc/", "/var/lib/kubelet/", "/dev/", "/mnt/", "/home/keys/", "/run/", "/var/run/", "/var/lock/", "/lock"}
	ignorePaths = append(ignorePaths, middleware.GetIgnorePaths()...)
	for _, igPath := range ignorePaths {
		if strings.Index(path, igPath) == 0 || path == strings.TrimRight(igPath, "/") {
			return true
		}
	}
	return false
}

func getAttachments(path string) []string {
	attachments, err := middleware.Call("pool.dataset.attachments_with_path", path)
	if err == nil {
		attachmentsResults := attachments.([]interface{})
		var attachmentList []string
		for _, attachmentEntry := range attachmentsResults {
			serviceType := attachmentEntry.(map[string]interface{})["type"].(string)
			if (serviceType == "Chart Releases" || serviceType == "Kubernetes") && isIXVolumePath(path, middleware.GetRootDataset()) {
				continue
			}
			attachmentList = append(attachmentList, attachmentEntry.(map[string]interface{})["type"].(string))
		}
		return attachmentList
	}
	return nil
}

func attachedPathValidation(path string) error {
	attachmentsResults := getAttachments(path)
	if attachmentsResults != nil && len(attachmentsResults) > 0 {
		return errors.Errorf("Invalid mount path. %s. Following service(s) uses this path: `%s`.", path, strings.Join(attachmentsResults[:], ", "))
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

func ixMountValidation(path string, pathToBeIgnored bool) error {
	pathList := pathToList(path)
	if pathToBeIgnored {
		// path list can be 0 if the path here was "/"
		if len(pathList) != 0 && len(pathList) < 3 && pathList[0] == "mnt" {
			return errors.Errorf("Invalid path %s. Mounting root dataset or path outside a pool is not allowed", path)
		}
		return nil
	} else if pathList[0] == "cluster" {
		clusterBlockPath := []string{"/cluster/ctdb_shared_vol", "/cluster"}
		for _, blPath := range clusterBlockPath {
			blPathLis := pathToList(blPath)
			if reflect.DeepEqual(pathList, blPathLis) {
				return errors.Errorf("Path %s is blocked and cannot be mounted.", path)
			}
		}
		return nil
	}
	return errors.Errorf("%s path not allowed to be mounted", path)
}
