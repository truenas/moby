package middleware

import (
	"bytes"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"os/exec"
)

func Call(method string, params ...interface{}) (interface{}, error) {
	var args []string
	args = append(args, "call")
	args = append(args, method)
	for _, entry := range params {
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
		return nil, err
	}
	var sanitizedResult interface{}
	// booleans are not json dumped right now by middleware client
	if string(out) == "True\n" || string(out) == "False\n" {
		out = bytes.ToLower(out)
	}

	err = json.Unmarshal([]byte(out), &sanitizedResult)
	if err != nil {
		sanitizedResult = string(out)
		logrus.Errorf("Failed to unmarshall middleware response for %s method with response %s: %s", method, out, err)
	}

	return sanitizedResult, err
}
