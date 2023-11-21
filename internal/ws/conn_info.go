package ws

import (
	"context"
	"errors"
	"github.com/gorilla/websocket"
	"log"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/ws/converters"
	"time"
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
	servDataChan chan<- session.ServerTx

	// msgToClientChan is the channel for messages ready to send to client
	msgToClientChan chan<- ws.RespMessage

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
					joinedMsg.MsgId = &refID

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

			// TODO: close connection
			return
		}
		msg, err := ws.ParseMessage(ctx, bytes)
		if err != nil {
			err = ws.ParseErrorToMessageError(err)
			var errDto *ws.Error
			errors.As(err, &errDto)
			rspMessage := ws.MessageError{
				BaseMessage: genBaseMessage(&ws.MsgKindError),
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
		ctx = context.WithValue(ctx, msgIDKey, msg.MsgID())

		switch m := msg.(type) {
		case *ws.MessageJoin:
			playerID, err := c.manager.JoinSession(ctx, c.sid, c.client, *m.Nickname, c.servDataChan)
			if err != nil {
				var errMsg ws.MessageError
				switch {
				case errors.Is(err, session.ErrNoSession):
					errMsg = genMessageError(m.MsgId, ws.ErrSessionExpired, "no such session")
				case errors.Is(err, session.ErrGameInProgress):
					errMsg = genMessageError(m.MsgId, ws.ErrUnknownSession, "game in progress now, no clients accepted")
				case errors.Is(err, session.ErrClientBanned):
					errMsg = genMessageError(m.MsgId, ws.ErrUnknownSession, "unknown session in request")
				case errors.Is(err, session.ErrNicknameUsed):
					errMsg = genMessageError(m.MsgId, ws.ErrNicknameUsed, "nickname is already used")
				case errors.Is(err, session.ErrLobbyFull):
					errMsg = genMessageError(m.MsgId, ws.ErrLobbyFull, "lobby is full")
				default:
					errMsg = genMessageError(m.MsgId, ws.ErrInternal, "internal error occurred")
				}
				log.Printf("ConnInfo client: %v parse message err: %v (code `%v`)",
					c.client.UUID().String(), err.Error(), errMsg.Code)
				c.msgToClientChan <- &errMsg
				// TODO: close connections
				return
			}
			c.playerID = playerID
		}

	}
}

func (c *ConnInfo) Dispose() {
	close(c.servDataChan)
	close(c.msgToClientChan)
	_ = c.wsConn.Close()
}

func genBaseMessage(kind *ws.MessageKind) ws.BaseMessage {
	newMsgID := ws.GenerateNewMessageID()
	now := time.Now()
	return ws.BaseMessage{Kind: kind, MsgId: &newMsgID, Time: (*ws.Time)(&now)}
}

func genMessageError(refID *ws.MessageId, code ws.ErrorKind, msg string) ws.MessageError {
	return ws.MessageError{
		BaseMessage: genBaseMessage(&ws.MsgKindError),
		Error: ws.Error{
			RefId:   refID,
			Code:    code,
			Message: msg,
		},
	}
}
