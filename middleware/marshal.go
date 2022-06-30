package middleware

import (
	"encoding/json"
)

func HandleMapUnmarshal(data string, mp *map[string]interface{}) error {
	err := json.Unmarshal([]byte(data), &mp)
	if err != nil {
		return err
	}
	return nil
}
