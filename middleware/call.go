package middleware

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"os/exec"
)

func Call(method string, params ...interface{}) (interface{}, error) {
	var args []string
	args = append(args, method)
	for _, entry := range params[1:] {
		sanitized, err := json.Marshal(entry)
		if err != nil {
			logrus.Errorf("Failed to marshal parameters for middleware: %s", err)
			return nil, err
		}
		args = append(args, string(sanitized[:]))
	}
	out, err := exec.Command(middlewareClientPath, args...).Output()
	if err != nil {
		logrus.Errorf("Middleware call to %s failed: %s", method, err)
	}
	var sanitizedResult []interface{}
	err = json.Unmarshal([]byte(out), &sanitizedResult)
	if err != nil {
		logrus.Errorf("Failed to unmarshall middleware response for %s method: %s", method, err)
	}
	return sanitizedResult, err
}
