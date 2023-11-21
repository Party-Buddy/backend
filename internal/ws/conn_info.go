package ws

import (
	"context"
	"errors"
	"github.com/gorilla/websocket"
	"log"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
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
				switch msg.(type) {
				case *session.MsgJoined:
					joinedServ := msg.(*session.MsgJoined)
					joinedMsg, err := MsgJoined2MessageJoined(*joinedServ)
					if err != nil {
						// TODO: handle error (means that something went wrong during converting)
					}
					newMsgId := ws.GenerateNewMessageID()
					joinedMsg.MsgId = &newMsgId
					refID := msgIDFromContext(joinedServ.Context())
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
			// TODO: ?
			continue
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
			continue
		}

		// TODO: get state
		// TODO: check message type availability for the state
		ctx = context.WithValue(ctx, msgIDKey, msg.MsgID())

		switch msg.(type) {
		case *ws.MessageJoin:
			joinMsg := msg.(*ws.MessageJoin)
			playerID, err := c.manager.JoinSession(ctx, c.sid, c.client, *joinMsg.Nickname, c.servDataChan)
			if err != nil {
				var errMsg ws.MessageError
				switch {
				case errors.Is(err, session.ErrNoSession):
					errMsg = genMessageError(joinMsg.MsgId, ws.ErrSessionExpired, "no such session")
				case errors.Is(err, session.ErrGameInProgress):
					errMsg = genMessageError(joinMsg.MsgId, ws.ErrUnknownSession, "game in progress now, no clients accepted")
				case errors.Is(err, session.ErrClientBanned):
					errMsg = genMessageError(joinMsg.MsgId, ws.ErrUnknownSession, "unknown session in request")
				case errors.Is(err, session.ErrNicknameUsed):
					errMsg = genMessageError(joinMsg.MsgId, ws.ErrNicknameUsed, "nickname is already used")
				case errors.Is(err, session.ErrLobbyFull):
					errMsg = genMessageError(joinMsg.MsgId, ws.ErrLobbyFull, "lobby is full")
				default:
					errMsg = genMessageError(joinMsg.MsgId, ws.ErrInternal, "internal error occurred")
				}
				log.Printf("ConnInfo client: %v parse message err: %v (code `%v`)",
					c.client.UUID().String(), err.Error(), errMsg.Code)
				c.msgToClientChan <- &errMsg
				log.Printf("ConnInfo client: %v parse message err: %v (code _)",
					c.client.UUID().String(), err.Error())
				continue
			}
			c.playerID = playerID
		}

	}
}

func (c *ConnInfo) Dispose() {
	close(c.servDataChan)
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