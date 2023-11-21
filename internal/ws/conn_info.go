package ws

import (
	"context"
	"errors"
	"github.com/gorilla/websocket"
	"log"
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
	log.Printf("ConnInfo start serving for client: %v", c.client.UUID().String())
	c.servDataChan = make(chan session.ServerTx)
	c.msgToClientChan = make(chan ws.RespMessage)
	go c.runReader(ctx)
	go c.runWriter(ctx)
	log.Printf("ConnInfo start serving for client: %v", c.client.UUID().String())
}

func (c *ConnInfo) runWriter(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-c.servDataChan:
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
					refID := joinMsgIDFromContext(joinedServ.Context())
					joinedMsg.MsgId = &refID

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

type joinKeyType int

var joinKey joinKeyType

func joinMsgIDFromContext(ctx context.Context) ws.MessageId {
	return ctx.Value(joinKey).(ws.MessageId)
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
		switch msg.(type) {
		case *ws.MessageJoin:
			joinMsg := msg.(*ws.MessageJoin)
			ctx = context.WithValue(ctx, joinKey, *joinMsg.MsgId)
			playerID, err := c.manager.JoinSession(ctx, c.sid, c.client, *joinMsg.Nickname, c.servDataChan)
			if err != nil {
				switch {
				case errors.Is(err, session.ErrNoSession):
					{
						errMsg := ws.MessageError{
							BaseMessage: genBaseMessage(&ws.MsgKindError),
							Error: ws.Error{
								RefId:   joinMsg.MsgId,
								Code:    ws.ErrUnknownSession,
								Message: err.Error(),
							},
						}
						log.Printf("ConnInfo client: %v parse message err: %v (code `%v`)",
							c.client.UUID().String(), err.Error(), errMsg.Code)
						c.msgToClientChan <- &errMsg
					}
				case errors.Is(err, session.ErrGameInProgress):
					{
						errMsg := ws.MessageError{
							BaseMessage: genBaseMessage(&ws.MsgKindError),
							Error: ws.Error{
								RefId:   joinMsg.MsgId,
								Code:    ws.ErrUnknownSession,
								Message: err.Error(),
							},
						}
						log.Printf("ConnInfo client: %v parse message err: %v (code `%v`)",
							c.client.UUID().String(), err.Error(), errMsg.Code)
						c.msgToClientChan <- &errMsg
					}
				case errors.Is(err, session.ErrClientBanned):
					{
						errMsg := ws.MessageError{
							BaseMessage: genBaseMessage(&ws.MsgKindError),
							Error: ws.Error{
								RefId:   joinMsg.MsgId,
								Code:    ws.ErrUnknownSession,
								Message: err.Error(),
							},
						}
						log.Printf("ConnInfo client: %v parse message err: %v (code `%v`)",
							c.client.UUID().String(), err.Error(), errMsg.Code)
						c.msgToClientChan <- &errMsg
					}
				case errors.Is(err, session.ErrNicknameUsed):
					{
						errMsg := ws.MessageError{
							BaseMessage: genBaseMessage(&ws.MsgKindError),
							Error: ws.Error{
								RefId:   joinMsg.MsgId,
								Code:    ws.ErrNicknameUsed,
								Message: err.Error(),
							},
						}
						log.Printf("ConnInfo client: %v parse message err: %v (code `%v`)",
							c.client.UUID().String(), err.Error(), errMsg.Code)
						c.msgToClientChan <- &errMsg
					}
				case errors.Is(err, session.ErrLobbyFull):
					{
						errMsg := ws.MessageError{
							BaseMessage: genBaseMessage(&ws.MsgKindError),
							Error: ws.Error{
								RefId:   joinMsg.MsgId,
								Code:    ws.ErrLobbyFull,
								Message: err.Error(),
							},
						}
						log.Printf("ConnInfo client: %v parse message err: %v (code `%v`)",
							c.client.UUID().String(), err.Error(), errMsg.Code)
						c.msgToClientChan <- &errMsg
					}
				}
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
	// TODO: set Time
	newMsgID := ws.GenerateNewMessageID()
	return ws.BaseMessage{Kind: kind, MsgId: &newMsgID}
}
