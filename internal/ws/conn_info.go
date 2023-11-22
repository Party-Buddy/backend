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

	// msgToClientChan is the channel for messages ready to send to client
	msgToClientChan chan<- ws.RespMessage

	// playerID is the player identifier inside the game
	playerID *session.PlayerId

	// msgID is used for getting new msg-id
	// DO NOT get the data by accessing field
	// use nextMsgID instead
	msgID atomic.Uint32

	// stopRequested indicates that wsConn and channels should be closed
	stopRequested bool

	// cancel is a function to call for cancelling runWriter, runServeToWriterConverter
	cancel context.CancelFunc
}

func NewConnInfo(
	manager *session.Manager,
	wsConn *websocket.Conn,
	clientId session.ClientId,
	sid session.SessionId) *ConnInfo {

	return &ConnInfo{
		manager:       manager,
		wsConn:        wsConn,
		client:        clientId,
		sid:           sid,
		msgID:         atomic.Uint32{},
		stopRequested: false,
	}
}

func (c *ConnInfo) StartReadAndWriteConn(ctx context.Context) {
	servChan := make(chan session.ServerTx)
	msgChan := make(chan ws.RespMessage)
	c.msgToClientChan = msgChan
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	go c.runReader(ctx, servChan)
	go c.runServeToWriterConverter(ctx, msgChan, servChan)
	go c.runWriter(ctx, msgChan)
	log.Printf("ConnInfo start serving for client: %v", c.client.UUID().String())
}

func (c *ConnInfo) runServeToWriterConverter(
	ctx context.Context,
	msgChan chan<- ws.RespMessage,
	servChan <-chan session.ServerTx) {
	for !c.stopRequested {
		select {
		case <-ctx.Done():
			close(msgChan)
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

					msgChan <- &joinedMsg

					// case *session.MsgDisconnect:
					//	c.StopRequested = true
					//	msgChan <- convert(m)
					//  close(msgChan)
					//  return
				}

			}
		}
	}
}

func (c *ConnInfo) runWriter(ctx context.Context, msgChan <-chan ws.RespMessage) {
	for !c.stopRequested {
		select {
		case <-ctx.Done():
			_ = c.wsConn.Close()
			return

		case msg := <-msgChan:
			{
				msg.SetMsgID(ws.MessageId(c.nextMsgID()))
				_ = c.wsConn.WriteJSON(msg)

				if c.stopRequested {
					_ = c.wsConn.Close()
					c.cancel()
					return
				}
			}
		}
	}
}

type msgIDKeyType int

var msgIDKey msgIDKeyType

func msgIDFromContext(ctx context.Context) ws.MessageId {
	return ctx.Value(msgIDKey).(ws.MessageId)
}

func (c *ConnInfo) runReader(ctx context.Context, servDataChan session.TxChan) {
	for !c.stopRequested {
		_, bytes, err := c.wsConn.ReadMessage()
		if err != nil {
			log.Printf("ConnInfo client: %v err: %v", c.client.UUID().String(), err.Error())

			if !c.stopRequested {
				c.Dispose()
			}
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

			c.Dispose()
			return
		}

		// TODO: get state
		// TODO: check message type availability for the state
		ctx = context.WithValue(ctx, msgIDKey, msg.GetMsgID())

		switch m := msg.(type) {
		case *ws.MessageJoin:
			c.handleJoin(ctx, m, servDataChan)
		}

	}
}

func (c *ConnInfo) Dispose() {
	log.Printf("ConnInfo client: %v disconnecting", c.client.UUID().String())
	c.stopRequested = true
	if c.playerID != nil { // playerID indicates that client has already joined
		// so we are asking manager to disconnect us
		// TODO: ask manager to disconnect self because player has already joined the session
		// c.manager.RequestDisconnect(ctx, c.sid, c.clientID, c.playerID)
	} else {
		c.cancel()
	}
}

func (c *ConnInfo) nextMsgID() uint32 {
	return c.msgID.Add(1)
}
