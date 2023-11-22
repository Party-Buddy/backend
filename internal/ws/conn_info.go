package ws

import (
	"context"
	"errors"
	"github.com/gorilla/websocket"
	"log"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/ws/converters"
	"party-buddy/internal/ws/utils"
	"sync/atomic"
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
	servDataChan session.TxChan

	// msgToClientChan is the channel for messages ready to send to client
	msgToClientChan chan<- ws.RespMessage

	// playerID is the player identifier inside the game
	playerID session.PlayerId

	// msgID is used for getting new msg-id
	// DO NOT get the data by accessing field
	// use nextMsgID instead
	msgID atomic.Uint32
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
		msgID:   atomic.Uint32{},
	}
}

func (c *ConnInfo) StartReadAndWriteConn(ctx context.Context) {
	servChan := make(chan session.ServerTx)
	c.servDataChan = servChan
	msgChan := make(chan ws.RespMessage)
	c.msgToClientChan = msgChan
	go c.runReader(ctx)
	go c.runServeToWriterConverter(ctx, msgChan, servChan)
	go c.runWriter(ctx, msgChan)
	log.Printf("ConnInfo start serving for client: %v", c.client.UUID().String())
}

func (c *ConnInfo) runServeToWriterConverter(
	ctx context.Context,
	msgChan chan<- ws.RespMessage,
	servChan <-chan session.ServerTx) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-servChan:
			{
				// TODO: ServeTx -> RespMessage
				// TODO: send converted msg to c.msgToClientChan
				switch m := msg.(type) {
				case *session.MsgJoined:
					joinedMsg := converters.ToMessageJoined(*m)
					refID := msgIDFromContext(m.Context())
					joinedMsg.RefID = &refID
					id := ws.MessageId(c.nextMsgID())
					joinedMsg.MsgId = &id

					msgChan <- &joinedMsg
				}
			}
		}
	}
}

func (c *ConnInfo) runWriter(ctx context.Context, msgChan <-chan ws.RespMessage) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-msgChan:
			{
				msg.SetMsgID(ws.MessageId(c.nextMsgID()))
				_ = c.wsConn.WriteJSON(msg)
			}
		}
	}
}

type msgIDKeyType int

var msgIDKey msgIDKeyType

func msgIDFromContext(ctx context.Context) ws.MessageId {
	return ctx.Value(msgIDKey).(ws.MessageId)
}

func (c *ConnInfo) runReader(ctx context.Context) {
	for {
		_, bytes, err := c.wsConn.ReadMessage()
		if err != nil {
			log.Printf("ConnInfo client: %v err: %v", c.client.UUID().String(), err.Error())

			// TODO: close connections
			return
		}
		msg, err := ws.ParseMessage(ctx, bytes)
		if err != nil {
			err = ws.ParseErrorToMessageError(err)
			var errDto *ws.Error
			errors.As(err, &errDto)
			rspMessage := ws.MessageError{
				BaseMessage: utils.GenBaseMessage(&ws.MsgKindError),
				Error:       *errDto,
			}
			log.Printf("ConnInfo client: %v parse message err: %v (code `%v`)",
				c.client.UUID().String(), errDto.Message, errDto.Code)
			c.msgToClientChan <- &rspMessage

			// TODO: close connections
			return
		}

		// TODO: get state
		// TODO: check message type availability for the state
		ctx = context.WithValue(ctx, msgIDKey, msg.GetMsgID())

		switch m := msg.(type) {
		case *ws.MessageJoin:
			c.handleJoin(ctx, m)
		}

	}
}

func (c *ConnInfo) Dispose() {
	close(c.msgToClientChan)
	_ = c.wsConn.Close()
}

func (c *ConnInfo) nextMsgID() uint32 {
	return c.msgID.Add(1)
}
