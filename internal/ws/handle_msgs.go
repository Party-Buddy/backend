package ws

import (
	"context"
	"errors"
	"log"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/ws/utils"
)

func (c *ConnInfo) handleJoin(ctx context.Context, m *ws.MessageJoin, servDataChan session.TxChan) {
	player, err := c.manager.JoinSession(ctx, c.sid, c.client, *m.Nickname, servDataChan)
	if err != nil {
		var errMsg ws.MessageError
		switch {
		case errors.Is(err, session.ErrNoSession):
			errMsg = utils.GenMessageError(m.MsgID, ws.ErrSessionExpired, "no such session")
		case errors.Is(err, session.ErrGameInProgress):
			errMsg = utils.GenMessageError(m.MsgID, ws.ErrUnknownSession, "game in progress now, no clients accepted")
		case errors.Is(err, session.ErrClientBanned):
			errMsg = utils.GenMessageError(m.MsgID, ws.ErrUnknownSession, "unknown session in request")
		case errors.Is(err, session.ErrNicknameUsed):
			errMsg = utils.GenMessageError(m.MsgID, ws.ErrNicknameUsed, "nickname is already used")
		case errors.Is(err, session.ErrLobbyFull):
			errMsg = utils.GenMessageError(m.MsgID, ws.ErrLobbyFull, "lobby is full")
		default:
			errMsg = utils.GenMessageError(m.MsgID, ws.ErrInternal, "internal error occurred")
		}
		log.Printf("ConnInfo client: %v parse message err: %v (code `%v`)",
			c.client.UUID().String(), err.Error(), errMsg.Code)
		c.msgToClientChan <- &errMsg

		c.dispose(ctx)
		return
	}
	c.playerID = &player.ID
}
