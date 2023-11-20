package ws

import (
	"context"
	"github.com/gorilla/websocket"
	"party-buddy/internal/session"
)

type ConnInfo struct {
	manager *session.Manager

	// wsConn is a WebSocket (ws) connection
	wsConn *websocket.Conn

	// client is the SessionId to which ws connection is related
	client session.ClientId

	// sid is the SessionId to which ws connection is related
	sid session.SessionId

	// servDataChan is the channel for getting message from server
	servDataChan chan session.ServerTx
}

func NewConnInfo(
	manager *session.Manager,
	wsConn *websocket.Conn,
	clientId session.ClientId,
	sid session.SessionId) *ConnInfo {

	return &ConnInfo{
		manager: manager,
		wsConn:  wsConn,
		client:  clientId,
		sid:     sid,
	}
}

func (c *ConnInfo) StartReadAndWriteConn(ctx context.Context) {
	ch := make(chan session.ServerTx)
	c.servDataChan = ch
	go c.runReader(ctx)
	go c.runWriter(ctx)
}

func (c *ConnInfo) runWriter(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-c.servDataChan:
			{
				_ = c.wsConn.WriteJSON(msg)
			}
		}
	}
}

func (c *ConnInfo) runReader(ctx context.Context) {
	for {
		_, bytes, err := c.wsConn.ReadMessage()
		if err != nil {
			// TODO: ?
		}
		msg, err := ParseMessage(ctx, bytes)
		if err != nil {
			// TODO: send malformed message to writer
			continue
		}

		// TODO: remove
		msg.isRecvMessage()

		// TODO: get state
		// TODO: check message type availability for the state
	}
}

func (c *ConnInfo) Dispose() {
	close(c.servDataChan)
	_ = c.wsConn.Close()
}
