package middleware

import (
	"context"
	"nhooyr.io/websocket"
)

type Client struct {
	id           string
	username     string
	password     string
	msg          string
	conn         *websocket.Conn
	method       string
	params       string
	ctx          context.Context
	reInitialize bool
}
