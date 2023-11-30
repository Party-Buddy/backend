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
	"time"
)

type ConnInfo struct {
	manager *session.Manager

	// wsConn is a WebSocket (ws) connection
	wsConn *websocket.Conn

	// client is the ClientID to which ws connection is related
	client session.ClientID

	// sid is the SessionID to which ws connection is related
	sid session.SessionID

	// msgToClientChan is the channel for messages ready to send to client
	msgToClientChan chan<- ws.RespMessage

	// playerID is the player identifier inside the game
	playerID *session.PlayerID

	// msgID is used for getting new msg-id
	// DO NOT get the data by accessing field
	// use nextMsgID instead
	msgID atomic.Uint32

	// stopRequested indicates that wsConn and channels should be closed
	stopRequested atomic.Bool

	// cancel is a function to call for cancelling runWriter, runServeToWriterConverter
	cancel context.CancelFunc

	// servDataChan here for closing
	servDataChan session.TxChan
}

func NewConnInfo(
	manager *session.Manager,
	wsConn *websocket.Conn,
	clientID session.ClientID,
	sid session.SessionID) *ConnInfo {

	return &ConnInfo{
		manager:       manager,
		wsConn:        wsConn,
		client:        clientID,
		sid:           sid,
		msgID:         atomic.Uint32{},
		stopRequested: atomic.Bool{},
	}
}

func (c *ConnInfo) StartReadAndWriteConn(ctx context.Context) {
	c.stopRequested.Store(false)
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
	defer close(msgChan)

	for !c.stopRequested.Load() {
		select {
		case <-ctx.Done():
			return

		case msg := <-servChan:
			{
				if msg == nil {
					c.stopRequested.Store(true)
					return
				}
				// TODO: ServeTx -> RespMessage
				// TODO: send converted msg to c.msgToClientChan
				switch m := msg.(type) {
				case *session.MsgJoined:
					joinedMsg := converters.ToMessageJoined(*m)
					refID := msgIDFromContext(m.Context())
					joinedMsg.RefID = &refID
					msgChan <- &joinedMsg

				case *session.MsgTaskStart:
					taskStartMsg := converters.ToMessageTaskStart(*m)
					msgChan <- &taskStartMsg

				case *session.MsgTaskEnd:
					taskEndMsg := converters.ToMessageTaskEnd(*m)
					msgChan <- &taskEndMsg
				}

			}
		}
	}
}

func (c *ConnInfo) runWriter(ctx context.Context, msgChan <-chan ws.RespMessage) {
	defer properWSClose(c.wsConn)
	for !c.stopRequested.Load() {
		select {
		case <-ctx.Done():
			return

		case msg := <-msgChan:
			{
				if msg == nil {
					c.cancel()
					return
				}

				msg.SetMsgID(ws.MessageID(c.nextMsgID()))
				_ = c.wsConn.WriteJSON(msg)

				if c.stopRequested.Load() {
					c.cancel()
					return
				}
			}
		}
	}
}

type msgIDKeyType int

var msgIDKey msgIDKeyType

func msgIDFromContext(ctx context.Context) ws.MessageID {
	return ctx.Value(msgIDKey).(ws.MessageID)
}

func (c *ConnInfo) runReader(ctx context.Context, servDataChan session.TxChan) {
	defer c.wsConn.Close()
	for !c.stopRequested.Load() {
		_, bytes, err := c.wsConn.ReadMessage()
		if err != nil {
			log.Printf("ConnInfo client: %v err: %v", c.client.UUID().String(), err.Error())

			if !c.stopRequested.Load() {
				c.dispose(ctx)
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

			c.dispose(ctx)
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

func properWSClose(wsConn *websocket.Conn) {
	timeout := 10 * time.Second
	_ = wsConn.SetWriteDeadline(time.Now().Add(timeout))
	_ = wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(timeout)
	_ = wsConn.Close()

}

// dispose is used for closing ws connection and related channels.
// There 2 possible cases to call dispose:
//  1. reader call dispose and the client had NOT joined the session (so it has no PlayerID)
//  2. reader call dispose and client had joined the session
//
// Disconnecting because of server initiative is handled in runServeToWriterConverter
func (c *ConnInfo) dispose(ctx context.Context) {
	log.Printf("ConnInfo client: %v disconnecting", c.client.UUID().String())
	c.stopRequested.Store(true)
	if c.playerID != nil { // playerID indicates that client has already joined
		// Here we are asking manager to disconnect us
		log.Printf("ConnInfo client: %v player: %v request disconnection from manager",
			c.client.UUID().String(), c.playerID.UUID().ID())
		c.manager.RemovePlayer(ctx, c.sid, *c.playerID)
	} else {
		// Manager knows nothing about client, so we just stop threads
		c.cancel()
		close(c.servDataChan)
	}
}

func (c *ConnInfo) nextMsgID() uint32 {
	return c.msgID.Add(1)
}
