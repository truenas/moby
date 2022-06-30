package middleware

import "errors"

func Call(method string, params ...interface{}) (map[string]interface{}, error) {
	m := clientConfig.client
	resp, err := m.get(method, params...)
	return resp, err
}

func (m *Client) get(method string, params ...interface{}) (map[string]interface{}, error) {
	if m == nil {
		return nil, errors.New("client is not initialized")
	}
	data := map[string]interface{}{
		"id":     m.id,
		"msg":    m.msg,
		"method": method,
		"params": params,
	}
	connResp, connErr := handleSocketCommunication(m.ctx, m.conn, data)
	return connResp, connErr
}

func testConnection() error {
	call, errs := Call("core.ping")
	if errs != nil {
		return errs
	}
	pong, ok := call["result"].(string)
	if !(ok) && pong != "pong" {
		return errors.New("received invalid response from middleware")
	}
	return nil
}
