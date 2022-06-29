package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"

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
	shutdownLock sync.Mutex
	reInitialize bool
}

func InitializeMiddleware(ctx context.Context, username string, password string) {
	for {
		clientConfig.client.shutdownLock.Lock()
		if !IsClientInitialized() || testConnection() != nil {

			err := Initialize(ctx, username, password)
			if err != nil {
				logrus.Debug("Failed to initialize middleware")
				logrus.Debug(err)
			}
		}
		clientConfig.client.shutdownLock.Unlock()
		time.Sleep(60 * time.Second)
	}
}

func AcquireShutdownLock() {
	clientConfig.client.shutdownLock.Lock()
}

func GetLoggerFile() *os.File {
	openLogfile, err := os.OpenFile("/root/run.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil
	}
	return openLogfile
}

func HandleMapMarshal(data map[string]interface{}) ([]byte, error) {
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

func GenerateSession(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
	connectionRequest := map[string]interface{}{
		"msg":     "connect",
		"version": "1",
		"support": []string{"1"},
	}
	connResp, connErr := handleSocketCommunication(ctx, conn, connectionRequest)
	if connErr != nil {
		return nil, connErr
	}
	return connResp, nil
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

func SafeInitialize(ctx context.Context, username string, password string) error {
	DeInitialize()
	err := Initialize(ctx, username, password)
	if err != nil {
		DeInitialize()
	}
	return err
}

func Initialize(ctx context.Context, username string, password string) error {
	if clientConfig == nil {
		clientConfig = &config{}
		err := clientConfig.InitConfig()
		if err != nil {
			return err
		}
	}
	clientConfig.client = &Client{ctx: ctx, username: username, password: password}
	if clientConfig.verifyVolumes {
		connErr := generateSocket(ctx, clientConfig.socketUrl, username, password)
		if connErr != nil {
			return connErr
		}
		connCheckErr := testConnection()
		if connCheckErr != nil {
			return connCheckErr
		}
	}
	return nil
}

func DeInitialize() {
	if clientConfig != nil && clientConfig.client != nil {
		clientConfig.client.Close()
		clientConfig.client = nil
	}
}

func IsClientInitialized() bool {
	if clientConfig != nil && clientConfig.client != nil {
		return true
	}
	return false
}

func CanVerifyVolumes() (bool, error) {
	if !(IsClientInitialized()) {
		return clientConfig.verifyVolumes, errors.New("middleware could not be initialized")
	}
	return clientConfig.verifyVolumes, nil
}

func CanVerifyAttachPath() bool {
	return clientConfig.verifyAttachedPath
}

func CanVerifyLockedVolumes() bool {
	return clientConfig.verifyLockedPath
}

func GetIgnorePaths() []string {
	return clientConfig.ignorePaths
}

func GetRootDataset() string {
	return clientConfig.appsDataset
}

func generateSocket(ctx context.Context, socketUrl string, username string, password string) error {
	conn, _, connErr := websocket.Dial(ctx, socketUrl, nil)
	if connErr != nil {
		return connErr
	}
	conn.SetReadLimit(32769 * 10)
	connectionResp, connErr := GenerateSession(ctx, conn)
	if connErr != nil {
		return connErr
	}
	if (username != "") && (password != "") {
		_, loginErr := LoginSession(ctx, conn, connectionResp["session"].(string), username, password)
		if loginErr != nil {
			return loginErr
		}
	}
	clientConfig.client = &Client{
		id:       connectionResp["session"].(string),
		msg:      "method",
		ctx:      ctx,
		conn:     conn,
		username: username,
		password: password,
	}
	return nil
}

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

func (m *Client) Close() {
	if m.conn != nil {
		m.conn.Close(websocket.StatusNormalClosure, "")
	}
}
