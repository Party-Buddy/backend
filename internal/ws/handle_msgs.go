package ws

import (
	"context"
	"errors"
	"fmt"
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
		log.Printf("ConnInfo client: %s join session %s err: %v (code `%v`)",
			c.client, c.sid, err, errMsg.Code)
		c.msgToClientChan <- &errMsg

		c.dispose(ctx)
		return
	}
	c.playerID = &player.ID
}

func (c *ConnInfo) handleTaskAnswer(ctx context.Context, m *ws.MessageTaskAnswer) {
	if c.playerID == nil {
		log.Printf("ConnInfo client: %s not joined the session %s", c.client, c.sid)
		c.dispose(ctx)
		return
	}

	var err error
	if m.Answer == nil {
		err = c.manager.UpdatePlayerAnswer(ctx, c.sid, *c.playerID, nil, *m.Ready, *m.TaskIdx)
	} else {
		var answer session.TaskAnswer
		switch *m.Answer.Type {
		case ws.Text:
			answer = session.TextTaskAnswer(*m.Answer.Text)
		case ws.CheckedText:
			answer = session.CheckedTextAnswer(*m.Answer.Text)
		case ws.Option:
			answer = session.TextTaskAnswer(*m.Answer.Option)
		default:
			panic(fmt.Sprintf("unsupported answer type in ConnInfo with sid %s clientID %s playerID %s",
				c.sid, c.client, *c.playerID))
		}
		err = c.manager.UpdatePlayerAnswer(ctx, c.sid, *c.playerID, answer, *m.Ready, *m.TaskIdx)
	}
	if err != nil {
		var errMsg ws.MessageError
		switch {
		case errors.Is(err, session.ErrNoSession):
			errMsg = utils.GenMessageError(m.MsgID, ws.ErrSessionExpired, "no such session")
		case errors.Is(err, session.ErrNoPlayer):
			errMsg = utils.GenMessageError(m.MsgID, ws.ErrInternal,
				fmt.Sprintf("client %s is not player with id %v in session", c.client, c.playerID.UUID().ID()))
		case errors.Is(err, session.ErrNoTask):
			errMsg = utils.GenMessageError(m.MsgID, ws.ErrMalformedMsg,
				fmt.Sprintf("no task with idx %v", *m.TaskIdx))
		case errors.Is(err, session.ErrTypesTaskAndAnswerMismatch):
			errMsg = utils.GenMessageError(m.MsgID, ws.ErrMalformedMsg, "provided answer type do not match task type")
		}
		log.Printf("ConnInfo client: %s in session %s task answer err: %v (code `%v`)",
			c.client, c.sid, err, errMsg.Code)
		c.msgToClientChan <- &errMsg

		c.dispose(ctx)
	}
}
