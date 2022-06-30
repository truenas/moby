package middleware

import (
	"encoding/json"
	"errors"
)

func HandleMapMarshal(data []interface{}) ([]byte, error) {
	jsonByteData, err := json.Marshal(data)
	if err != nil {
		return nil, errors.New("can't parse map object")
	}
	return jsonByteData, err
}

func HandleMapUnmarshal(data string, mp *map[string]interface{}) error {
	err := json.Unmarshal([]byte(data), &mp)
	if err != nil {
		return err
	}
	return nil
}
