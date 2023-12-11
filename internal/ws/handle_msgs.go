package ws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/ws/converters"
	"party-buddy/internal/ws/utils"
)

func (c *ConnInfo) handleJoin(ctx context.Context, m *ws.MessageJoin, servDataChan session.TxChan) {
	player, err := c.manager.JoinSession(ctx, c.sid, c.client, *m.Nickname, servDataChan)
	if err != nil {
		code, message := converters.ErrorCodeAndMessage(err)
		errMsg := utils.GenMessageError(m.MsgID, code, message)
		log.Printf("ConnInfo client: %s join session %s err: %v (code `%v`)",
			c.client, c.sid, err, errMsg.Code)
		c.msgToClientChan <- &errMsg

		c.dispose(ctx)
		return
	}
	c.playerID = &player.ID
}

func (c *ConnInfo) handleReady(ctx context.Context, m *ws.MessageReady) {
	if c.playerID == nil {
		log.Panicf("ConnInfo client: %s is not part of the session %s", c.client, c.sid)
		return
	}

	err := c.manager.SetPlayerReady(ctx, c.sid, *c.playerID, *m.Ready)
	if err != nil {
		code, message := converters.ErrorCodeAndMessage(err)
		errMsg := utils.GenMessageError(m.MsgID, code, message)
		log.Printf("ConnInfo client: %s in session %s ready err: %s (code `%s`)",
			c.client, c.sid, err, errMsg.Code)
		c.msgToClientChan <- &errMsg

		c.dispose(ctx)
	}
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
		var code ws.ErrorKind
		var message string

		switch {
		case errors.Is(err, session.ErrTaskIndexOutOfBounds):
			code, message = ws.ErrMalformedMsg, fmt.Sprintf("the task index %d is out of bounds", *m.TaskIdx)
		default:
			code, message = converters.ErrorCodeAndMessage(err)
		}

		errMsg := utils.GenMessageError(m.MsgID, code, message)
		log.Printf("ConnInfo client: %s in session %s task answer err: %v (code `%v`)",
			c.client, c.sid, err, errMsg.Code)
		c.msgToClientChan <- &errMsg

		c.dispose(ctx)
	}
}
