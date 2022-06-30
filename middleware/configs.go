package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
)

type config struct {
	socketUrl          string
	verifyVolumes      bool
	verifyLockedPath   bool
	verifyAttachedPath bool
	appsDataset        string
	ignorePaths        []string
}

func (c *config) loadConfig() (map[string]interface{}, error) {
	var configPath = fmt.Sprintf("%s/%s", configDir, configFile)
	data, err := os.ReadFile(configPath)

	if err != nil {
		return nil, err
	}
	configMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(data), &configMap)
	if err != nil {
		logrus.Errorf("Failed to load configuration for middleware: %s", err)
	}
	return configMap, err
}

func parseValue(name string, configMap map[string]interface{}, defaultValue interface{}) interface{} {
	value, ok := configMap[name]
	if ok {
		return value
	}
	return defaultValue
}

func parseBoolValue(name string, configMap map[string]interface{}, defaultValue bool) bool {
	value, ok := parseValue(name, configMap, defaultValue).(bool)
	if ok {
		return value
	}
	return defaultValue
}

func parseStringListValue(name string, configMap map[string]interface{}, defaultValue []string) []string {
	value, ok := parseValue(name, configMap, defaultValue).([]interface{})
	if ok {
		var stringList []string
		for _, val := range value {
			strVal, ok := val.(string)
			if ok {
				stringList = append(stringList, strVal)
			}
		}
		return stringList
	}
	return defaultValue
}

func InitConfig() error {
	clientConfig = &config{}
	configMap, err := clientConfig.loadConfig()
	if err != nil {
		return err
	}
	value, ok := configMap["socketUrl"]
	if ok {
		clientConfig.socketUrl = value.(string)
	} else {
		return errors.New("socketURL must be specified")
	}
	value, ok = configMap["appsDataset"]
	if ok {
		clientConfig.appsDataset = value.(string)
	} else {
		return errors.New("apps (ix-applications) dataset complete name must be specified i.e tank/ix-applications")
	}

	clientConfig.verifyVolumes = parseBoolValue("verifyVolumes", configMap, true)
	clientConfig.verifyLockedPath = parseBoolValue("verifyLockedPath", configMap, true)
	clientConfig.verifyAttachedPath = parseBoolValue("verifyAttachedPath", configMap, true)
	clientConfig.ignorePaths = parseStringListValue("ignorePaths", configMap, []string{})
	return nil
}

func CanVerifyVolumes() (bool, error) {
	return clientConfig.verifyVolumes, nil
}

func CanVerifyAttachPath() bool {
	return clientConfig.verifyAttachedPath
}

func CanVerifyLockedVolumes() bool {
	return clientConfig.verifyLockedPath
}

func GetIgnorePaths() []string {
	return clientConfig.ignorePaths
}

func GetRootDataset() string {
	return clientConfig.appsDataset
}
