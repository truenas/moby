package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"os"
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
	reInitialize bool
}

func InitializeMiddleware(ctx context.Context, username string, password string) {
	for {
		// FIXME: Let's fix the shutdown lock logic
		// shutdownLock.Lock()
		if !IsClientInitialized() || testConnection() != nil {
			DeInitialize()
			err := Initialize(ctx, username, password)
			if err != nil {
				logrus.Debug("Failed to initialize middleware")
				logrus.Debug(err)
			}
		}
		// shutdownLock.Unlock()
		time.Sleep(60 * time.Second)
	}
}

func AcquireShutdownLock() {
	shutdownLock.Lock()
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
