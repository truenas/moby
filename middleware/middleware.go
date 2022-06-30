package middleware

import (
	"context"
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

func (m *Client) Close() {
	if m.conn != nil {
		m.conn.Close(websocket.StatusNormalClosure, "")
	}
}
