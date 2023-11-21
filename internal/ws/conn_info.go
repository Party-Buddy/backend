package ws

import (
	"context"
	"github.com/gorilla/websocket"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
)

type ConnInfo struct {
	manager *session.Manager

	// wsConn is a WebSocket (ws) connection
	wsConn *websocket.Conn

	// client is the ClientId to which ws connection is related
	client session.ClientId

	// sid is the SessionId to which ws connection is related
	sid session.SessionId

	// servDataChan is the channel for getting event messages from server
	servDataChan chan session.ServerTx

	// msgToClientChan is the channel for messages ready to send to client
	msgToClientChan chan ws.RespMessage

	// playerID is the player identifier inside the game
	playerID session.PlayerId
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
	c.servDataChan = make(chan session.ServerTx)
	c.msgToClientChan = make(chan ws.RespMessage)
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
				// TODO: ServeTx -> BaseMessage
				// TODO: send converted msg to c.msgToClientChan
				switch msg.(type) {
				case *session.MsgJoined:
					joinedServ := msg.(*session.MsgJoined)
					joinedMsg, err := MsgJoined2MessageJoined(*joinedServ)
					if err != nil {

					}
					c.msgToClientChan <- &joinedMsg
				}
			}

		case msg := <-c.msgToClientChan:
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
			continue
		}
		msg, err := ws.ParseMessage(ctx, bytes)
		if err != nil {
			// TODO: send malformed message to writer
			continue
		}

		// TODO: get state
		// TODO: check message type availability for the state
		switch msg.(type) {
		case *ws.MessageJoin:
			joinMsg := msg.(*ws.MessageJoin)
			playerID, err := c.manager.JoinSession(ctx, c.sid, c.client, *joinMsg.Nickname, c.servDataChan)
			if err != nil {
				// TODO: handle errors
			}
			c.playerID = playerID
		}

	}
}

func (c *ConnInfo) Dispose() {
	close(c.servDataChan)
	_ = c.wsConn.Close()
}
