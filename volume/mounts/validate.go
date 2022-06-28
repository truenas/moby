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

func lockedpathValidation(path string) error {
	call, err := middleware.Call("pool.dataset.path_in_locked_datasets", path)
	if err == nil {
		islocked, ok := call["result"].(bool)
		if ok && islocked {
			return errors.Errorf("Dataset path is locked")
		}
	}
	return nil
}

func isIXvolumePath(path string, dataset_path string) bool {
	release_path := filepath.Join("mnt", dataset_path, "releases")
	if strings.Index(path, "/"+release_path) == 0 {
		app_path := strings.Replace(path, "/"+release_path+"/", "", 1)
		app_name := strings.Split(app_path, "/")[0]
		volume_path := filepath.Join(release_path, app_name, "volumes", "ix_volumes")
		if strings.Contains(path, "/"+volume_path) {
			return true
		}
	}
	return false
}

func ignorePath(path string, dataset string) bool {
	ignore_path := []string{"/etc/", "/sys/", "/proc/", "/var/lib/kubelet/pods/"}
	ignore_path = append(ignore_path, middleware.GetIgnorePaths()...)
	for _, ig_path := range ignore_path {
		if strings.Index(path, ig_path) == 0 {
			return true
		}
	}
	if isIXvolumePath(path, dataset) {
		return true
	}
	return false
}

func getAttachments(path string) []string {
	attachments, err := middleware.Call("pool.dataset.attachments_with_path", path)
	if err == nil {
		attachments_results := attachments["result"].([]interface{})
		fmt.Println("results:  ", attachments_results)
		attachment_list := []string{}
		for _, attach := range attachments_results {
			attach_names := attach.(map[string]interface{})["attachments"].([]interface{})
			for _, name := range attach_names {
				attachment_list = append(attachment_list, name.(string))
			}
		}
		return attachment_list
	}
	return nil
}

func attachedPathValidation(path string) error {
	dataset_path := middleware.GetRootDataset()
	if ignorePath(path, dataset_path) {
		return nil
	}
	attachments_results := getAttachments(path)
	if attachments_results != nil && len(attachments_results) > 0 {
		attach := ""
		for _, pa := range attachments_results {
			attach += pa + ","
		}
		return errors.Errorf("Invalid mount path. %s. Following app uses this path. `%s`.", path, attach)
	}
	return nil
}

func pathToList(path string) []string {
	raw_path_list := strings.Split(path, "/")
	process_path_list := []string{}
	for _, name := range raw_path_list {
		if name != "" {
			process_path_list = append(process_path_list, name)
		}
	}
	return process_path_list
}

func ixMountValidation(path string) error {
	path_list := pathToList(path)
	block_path := []string{"/cluster/ctdb_shared_vol", "/cluster"}
	if len(path_list) < 3 && path_list[0] == "mnt" {
		return errors.Errorf("Invalid path %s. Mounting a pool is not allowed", path)
	}
	for _, bl_path := range block_path {
		bl_path_lis := pathToList(bl_path)
		if reflect.DeepEqual(path_list, bl_path_lis) {
			return errors.Errorf("Path %s is blocked.", path)
		}
	}
	return nil
}
