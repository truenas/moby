package middleware

import (
	"errors"
	"fmt"
	"os"
	"time"
)

var (
	configDir              = getConfigDir()
	configFile             = "docker_middleware.json"
	client_config  *config = nil
	time_out_limit         = 8 * time.Second
)

func getConfigDir() string {
	if os.Getenv("DOCKER_MIDDLEWARE_CONFIG_DIR") != "" {
		return os.Getenv("DOCKER_MIDDLEWARE_CONFIG_DIR")
	}
	return "/etc/default"
}

type config struct {
	socket_url           string
	client               *Client
	verify_volumes       bool
	verify_locked_path   bool
	verify_attached_path bool
	root_dataset         string
	ignore_paths         []string
}

func (c *config) loadConfig() (map[string]interface{}, error) {
	var config_path string = fmt.Sprintf("%s/%s", configDir, configFile)
	data, err := os.ReadFile(config_path)

	if err != nil {
		return nil, err
	}
	config_map := make(map[string]interface{})
	err = HandleMapUnmarshal(string(data), &config_map)
	if err != nil {
		return nil, err
	}
	return config_map, nil
}

func parseValue(name string, config_map map[string]interface{}, default_value interface{}) interface{} {
	value, ok := config_map[name]
	if ok {
		return value
	}
	return default_value
}

func parseBoolValue(name string, config_map map[string]interface{}, default_value bool) bool {
	value, ok := parseValue(name, config_map, default_value).(bool)
	if ok {
		return value
	}
	return default_value
}

func parseStringListValue(name string, config_map map[string]interface{}, default_value []string) []string {
	value, ok := parseValue(name, config_map, default_value).([]interface{})
	if ok {
		string_list := []string{}
		for _, val := range value {
			str_val, ok := val.(string)
			if ok {
				string_list = append(string_list, str_val)
			}
		}
		return string_list
	}
	return default_value
}

func (c *config) InitConfig() error {
	config_map, err := c.loadConfig()
	if err != nil {
		return err
	}
	value, ok := config_map["socket_url"]
	if ok {
		c.socket_url = value.(string)
	} else {
		return errors.New("Invalid configuration. missing socket_url.")
	}
	value, ok = config_map["root_dataset"]
	if ok {
		c.root_dataset = value.(string)
	} else {
		return errors.New("Invalid configuration. root dataset for ix application is required.")
	}

	c.verify_volumes = parseBoolValue("verify_volumes", config_map, true)
	c.verify_locked_path = parseBoolValue("verify_locked_path", config_map, true)
	c.verify_attached_path = parseBoolValue("verify_attached_path", config_map, true)
	c.ignore_paths = parseStringListValue("ignore_paths", config_map, []string{})
	return nil
}
