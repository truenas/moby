package middleware

import (
	"context"
	"encoding/json"
	"errors"

	"nhooyr.io/websocket"
)

type Client struct {
	id       string
	username string
	password string
	msg      string
	conn     *websocket.Conn
	method   string
	params   string
	ctx      context.Context
}

func HandleMapMarshal(data map[string]interface{}) ([]byte, error) {
	json_byte_data, err := json.Marshal(data)
	if err != nil {
		return nil, errors.New("Cann't parsed map object")
	}
	return json_byte_data, err
}

func HandleMapUnmarshal(data string, mp *map[string]interface{}) error {
	err := json.Unmarshal([]byte(data), &mp)
	if err != nil {
		return err
	}
	return nil
}

func socketCommunication(ctx context.Context, conn *websocket.Conn,
	data map[string]interface{}, resp chan map[string]interface{}, err chan error) {
	if conn == nil {
		err <- errors.New("Socket is not initialzed")
		return
	}
	parsed_byte, errs := HandleMapMarshal(data)
	if errs != nil {
		err <- errs
		return
	}
	conn_write_err := conn.Write(ctx, 1, parsed_byte)
	if conn_write_err != nil {
		err <- conn_write_err
		return
	}
	_, conn_resp, conn_read_err := conn.Read(ctx)
	if conn_read_err != nil {
		err <- conn_read_err
		return
	}
	response := make(map[string]interface{})
	errs = HandleMapUnmarshal(string(conn_resp[:]), &response)
	if errs != nil {
		err <- errs
		return
	}
	resp <- response
}

func handleSocketCommunication(ctx context.Context, conn *websocket.Conn, data map[string]interface{}) (map[string]interface{}, error) {
	response := make(chan map[string]interface{})
	err := make(chan error)
	ctx, cancel := context.WithTimeout(ctx, time_out_limit)
	defer cancel()
	go socketCommunication(ctx, conn, data, response, err)
	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("Request time out error")
		case errs := <-err:
			return nil, errs
		case resp := <-response:
			return resp, nil
		}
	}
}

func GenerateSession(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
	connection_request := map[string]interface{}{
		"msg":     "connect",
		"version": "1",
		"support": []string{"1"},
	}
	conn_resp, conn_err := handleSocketCommunication(ctx, conn, connection_request)
	if conn_err != nil {
		return nil, conn_err
	}
	return conn_resp, nil
}

func LoginSession(ctx context.Context, conn *websocket.Conn, id string, username string, password string) (map[string]interface{}, error) {
	login_request := map[string]interface{}{
		"id":     id,
		"msg":    "method",
		"method": "auth.login",
		"params": []string{username, password},
	}
	conn_resp, conn_err := handleSocketCommunication(ctx, conn, login_request)
	if conn_err != nil {
		return nil, conn_err
	}
	if !conn_resp["result"].(bool) {
		return nil, errors.New("Invalid credentials")
	}

	return conn_resp, nil
}

func testConnection() error {
	call, errs := Call("core.ping")
	if errs != nil {
		return errs
	}
	pong, ok := call["result"].(string)
	if !(ok) && pong != "pong" {
		return errors.New("Invalid credentials")
	}
	return nil
}

func SafeInitialize(ctx context.Context, username string, password string) error {
	DeInitialize()
	err := Initialize(ctx, username, password)
	if err != nil {
		DeInitialize()
		return err
	}
	return nil
}

func defaultInitialize() error {
	err := SafeInitialize(context.Background(), "", "")
	if err != nil {
		return err
	}
	return nil
}

func Initialize(ctx context.Context, username string, password string) error {
	if client_config == nil {
		client_config = &config{}
		err := client_config.InitConfig()
		if err != nil {
			return err
		}
	}
	if !(client_config.verify_volumes) {
		client_config.client = &Client{ctx: ctx, username: username, password: password}
		return nil
	} else {
		conn_err := generateSocket(ctx, client_config.socket_url, username, password)
		if conn_err != nil {
			return conn_err
		}
		conn_check_err := testConnection()
		if conn_check_err != nil {
			return conn_check_err
		}
	}
	return nil
}

func DeInitialize() {
	if client_config != nil && client_config.client != nil {
		client_config.client.Close()
		client_config.client = nil
	}
}

func IsClientInitialized() bool {
	if client_config != nil && client_config.client != nil {
		return true
	} else if defaultInitialize() == nil {
		return true
	}
	return false
}

func CanVerifyVolumes() (bool, error) {
	if !(IsClientInitialized()) {
		return client_config.verify_volumes, errors.New("Middleware not initialize")
	}
	return client_config.verify_volumes, nil
}

func CanVerifyAttachPath() bool {
	return client_config.verify_attached_path
}

func CanVerifyLockedvolumes() bool {
	return client_config.verify_locked_path
}

func GetIgnorePaths() []string {
	return client_config.ignore_paths
}

func GetRootDataset() string {
	return client_config.root_dataset
}

func generateSocket(ctx context.Context, socket_url string, username string, password string) error {
	conn, _, conn_err := websocket.Dial(ctx, socket_url, nil)
	if conn_err != nil {
		return conn_err
	}
	conn.SetReadLimit(32769 * 10)
	connection_resp, com_err := GenerateSession(ctx, conn)
	if com_err != nil {
		return com_err
	}
	if (username != "") && (password != "") {
		_, login_err := LoginSession(ctx, conn, connection_resp["session"].(string), username, password)
		if login_err != nil {
			return login_err
		}
	}
	client_config.client = &Client{
		id:       connection_resp["session"].(string),
		msg:      "method",
		ctx:      ctx,
		conn:     conn,
		username: username,
		password: password,
	}
	return nil
}

func Call(method string, params ...interface{}) (map[string]interface{}, error) {
	m := client_config.client
	resp, err := m.get(method, params...)
	if err != nil {
		con_err := SafeInitialize(m.ctx, m.username, m.password)
		if con_err == nil {
			m = client_config.client
			resp, err = m.get(method, params...)
			return resp, err
		}
		return nil, err
	}
	return resp, nil
}

func (m *Client) get(method string, params ...interface{}) (map[string]interface{}, error) {
	if m == nil {
		return nil, errors.New("Client is not intialized")
	}
	data := map[string]interface{}{
		"id":     m.id,
		"msg":    m.msg,
		"method": method,
		"params": params,
	}
	conn_resp, conn_err := handleSocketCommunication(m.ctx, m.conn, data)
	if conn_err != nil {
		return nil, conn_err
	}
	return conn_resp, nil
}

func (m *Client) Close() {
	if m.conn != nil {
		m.conn.Close(websocket.StatusNormalClosure, "")
	}
}
