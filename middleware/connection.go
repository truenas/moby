package middleware

import (
	"context"
	"errors"

	"nhooyr.io/websocket"
)

func GenerateSession(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
	connectionRequest := map[string]interface{}{
		"msg":     "connect",
		"version": "1",
		"support": []string{"1"},
	}
	return handleSocketCommunication(ctx, conn, connectionRequest)
}

func LoginSession(ctx context.Context, conn *websocket.Conn, id string, username string, password string) (map[string]interface{}, error) {
	loginRequest := map[string]interface{}{
		"id":     id,
		"msg":    "method",
		"method": "auth.login",
		"params": []string{username, password},
	}
	connResp, connErr := handleSocketCommunication(ctx, conn, loginRequest)
	if connErr != nil {
		return nil, connErr
	}
	if !connResp["result"].(bool) {
		return nil, errors.New("invalid credentials")
	}

	return connResp, nil
}

func testConnection() error {
	call, errs := Call("core.ping")
	if errs != nil {
		return errs
	}
	pong, ok := call["result"].(string)
	if !(ok) && pong != "pong" {
		return errors.New("invalid credentials")
	}
	return nil
}

func socketCommunication(ctx context.Context, conn *websocket.Conn,
	data map[string]interface{}, resp chan map[string]interface{}, err chan error) {
	if conn == nil {
		err <- errors.New("socket is not initialized")
		return
	}
	parsedByte, errs := HandleMapMarshal(data)
	if errs != nil {
		err <- errs
		return
	}
	connWriteErr := conn.Write(ctx, 1, parsedByte)
	if connWriteErr != nil {
		err <- connWriteErr
		return
	}
	_, connResp, connReadErr := conn.Read(ctx)
	if connReadErr != nil {
		err <- connReadErr
		return
	}
	response := make(map[string]interface{})
	errs = HandleMapUnmarshal(string(connResp[:]), &response)
	if errs != nil {
		err <- errs
		return
	}
	resp <- response
}

func handleSocketCommunication(ctx context.Context, conn *websocket.Conn, data map[string]interface{}) (map[string]interface{}, error) {
	response := make(chan map[string]interface{})
	err := make(chan error)
	ctx, cancel := context.WithTimeout(ctx, timeoutLimit)
	defer cancel()
	go socketCommunication(ctx, conn, data, response, err)
	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("request time out error")
		case errs := <-err:
			return nil, errs
		case resp := <-response:
			return resp, nil
		}
	}
}
