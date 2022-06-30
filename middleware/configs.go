package middleware

import (
	"errors"
	"fmt"
	"os"
)

type config struct {
	socketUrl          string
	client             *Client
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
	err = HandleMapUnmarshal(string(data), &configMap)
	if err != nil {
		return nil, err
	}
	return configMap, nil
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

func (c *config) InitConfig() error {
	configMap, err := c.loadConfig()
	if err != nil {
		return err
	}
	value, ok := configMap["socketUrl"]
	if ok {
		c.socketUrl = value.(string)
	} else {
		return errors.New("socketURL must be specified")
	}
	value, ok = configMap["appsDataset"]
	if ok {
		c.appsDataset = value.(string)
	} else {
		return errors.New("apps (ix-applications) dataset complete name must be specified i.e tank/ix-applications")
	}

	c.verifyVolumes = parseBoolValue("verifyVolumes", configMap, true)
	c.verifyLockedPath = parseBoolValue("verifyLockedPath", configMap, true)
	c.verifyAttachedPath = parseBoolValue("verifyAttachedPath", configMap, true)
	c.ignorePaths = parseStringListValue("ignorePaths", configMap, []string{})
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
