package ws

import (
	"context"
	"errors"
	"log"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/ws/converters"
)

func (c *ConnInfo) handleJoin(ctx context.Context, m *ws.MessageJoin) {
	playerID, err := c.manager.JoinSession(ctx, c.sid, c.client, *m.Nickname, c.servDataChan)
	if err != nil {
		var errMsg ws.MessageError
		id := ws.MessageId(c.nextMsgID())
		switch {
		case errors.Is(err, session.ErrNoSession):
			errMsg = converters.GenMessageError(m.MsgId, ws.ErrSessionExpired, "no such session")
		case errors.Is(err, session.ErrGameInProgress):
			errMsg = converters.GenMessageError(m.MsgId, ws.ErrUnknownSession, "game in progress now, no clients accepted")
		case errors.Is(err, session.ErrClientBanned):
			errMsg = converters.GenMessageError(m.MsgId, ws.ErrUnknownSession, "unknown session in request")
		case errors.Is(err, session.ErrNicknameUsed):
			errMsg = converters.GenMessageError(m.MsgId, ws.ErrNicknameUsed, "nickname is already used")
		case errors.Is(err, session.ErrLobbyFull):
			errMsg = converters.GenMessageError(m.MsgId, ws.ErrLobbyFull, "lobby is full")
		default:
			errMsg = converters.GenMessageError(m.MsgId, ws.ErrInternal, "internal error occurred")
		}
		errMsg.MsgId = &id
		log.Printf("ConnInfo client: %v parse message err: %v (code `%v`)",
			c.client.UUID().String(), err.Error(), errMsg.Code)
		c.msgToClientChan <- &errMsg
		// TODO: close connections
		return
	}
	c.playerID = playerID
}
