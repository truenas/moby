package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
)

type config struct {
	verifyVolumes      bool
	verifyLockedPath   bool
	verifyAttachedPath bool
	appsDataset        string
	ignorePaths        []string
}

func loadConfig() (map[string]interface{}, error) {
	var configPath = fmt.Sprintf("%s/%s", configDir, configFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	configMap := make(map[string]interface{})
	err = json.Unmarshal(data, &configMap)
	if err != nil {
		logrus.Errorf("Failed to load configuration for middleware: %s", err)
	}
	return configMap, err
}

func parseValue(name string, configMap map[string]interface{}, defaultValue bool) bool {
	value, ok := configMap[name]
	if ok {
		return value.(bool)
	}
	return defaultValue
}

func parseStringListValue(name string, configMap map[string]interface{}, defaultValue []string) []string {
	value, ok := configMap[name]
	if ok {
		var stringList []string
		for _, val := range value.([]interface{}) {
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
	configMap, err := loadConfig()
	if err != nil {
		return err
	}
	requiredKeys := [2]string{"appsDataset", "verifyVolumes"}
	for _, key := range requiredKeys {
		if _, ok := configMap[key]; !ok {
			errString := fmt.Sprintf("%s key must be specified", key)
			return errors.New(errString)
		}
	}

	clientConfig = &config{}
	clientConfig.appsDataset = configMap["appsDataset"].(string)
	clientConfig.verifyVolumes = configMap["verifyVolumes"].(bool)
	clientConfig.verifyLockedPath = parseValue("verifyLockedPath", configMap, true)
	clientConfig.verifyAttachedPath = parseValue("verifyAttachedPath", configMap, true)
	clientConfig.ignorePaths = parseStringListValue("ignorePaths", configMap, []string{})
	return nil
}

func CanVerifyVolumes() bool {
	return clientConfig != nil && clientConfig.verifyVolumes
}

func CanVerifyAttachPath() bool {
	return clientConfig != nil && clientConfig.verifyAttachedPath
}

func CanVerifyLockedVolumes() bool {
	return clientConfig != nil && clientConfig.verifyLockedPath
}

func GetIgnorePaths() []string {
	return clientConfig.ignorePaths
}

func GetRootDataset() string {
	return clientConfig.appsDataset
}
