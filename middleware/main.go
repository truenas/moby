package middleware

import "time"

var (
	middlewareClientPath         = "midclt"
	configDir                    = "/etc/docker"
	configFile                   = "middleware.json"
	clientConfig         *config = nil
	timeoutLimit                 = 10 * time.Second
)
