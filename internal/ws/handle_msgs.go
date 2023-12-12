package ws

import (
	"context"
	"errors"
	"fmt"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/ws/converters"
	"party-buddy/internal/ws/utils"
)

func (c *Conn) handleJoin(ctx context.Context, m *ws.MessageJoin, servDataChan session.TxChan) {
	player, err := c.manager.JoinSession(ctx, c.sid, c.client, *m.Nickname, servDataChan)
	if err != nil {
		code, message := converters.ErrorCodeAndMessage(err)
		errMsg := utils.GenMessageError(m.MsgID, code, message)
		c.readerLog.Printf("the manager returned an error while processing the Join message: %s (code `%s`)",
			err, errMsg.Code)
		c.msgToClientChan <- &errMsg

		c.dispose(ctx)
		return
	}
	c.setPlayerID(player.ID)
}

func (c *Conn) handleReady(ctx context.Context, m *ws.MessageReady) {
	if c.playerID == nil {
		code, message := ws.ErrInternal, "internal error"
		errMsg := utils.GenMessageError(m.MsgID, code, message)
		c.readerLog.Printf("the client has not yet joined the session (code `%s`)", code)
		c.msgToClientChan <- &errMsg

		c.dispose(ctx)
		return
	}

	err := c.manager.SetPlayerReady(ctx, c.sid, *c.playerID, *m.Ready)
	if err != nil {
		code, message := converters.ErrorCodeAndMessage(err)
		errMsg := utils.GenMessageError(m.MsgID, code, message)
		c.readerLog.Printf("the manager returned an error while processing the Ready message: %s (code `%s`)",
			err, errMsg.Code)
		c.msgToClientChan <- &errMsg

		c.dispose(ctx)
	}
}

func (c *Conn) handleTaskAnswer(ctx context.Context, m *ws.MessageTaskAnswer) {
	if c.playerID == nil {
		code, message := ws.ErrInternal, "internal error"
		errMsg := utils.GenMessageError(m.MsgID, code, message)
		c.readerLog.Printf("the client has not yet joined the session (code `%s`)", code)
		c.msgToClientChan <- &errMsg

		c.dispose(ctx)
		return
	}

	var answer session.TaskAnswer
	if m.Answer != nil {
		switch *m.Answer.Type {
		case ws.Text:
			answer = session.TextTaskAnswer(*m.Answer.Text)
		case ws.CheckedText:
			answer = session.CheckedTextAnswer(*m.Answer.Text)
		case ws.Option:
			answer = session.ChoiceTaskAnswer(*m.Answer.Option)
		default:
			c.readerLog.Panicf("unsupported answer type: %s", *m.Answer.Type)
		}
	}

	err := c.manager.UpdatePlayerAnswer(ctx, c.sid, *c.playerID, answer, *m.Ready, *m.TaskIdx)

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
		c.readerLog.Printf("the manager returned an error while processing the TaskAnswer message: %s (code `%s`)",
			err, errMsg.Code)
		c.msgToClientChan <- &errMsg

		c.dispose(ctx)
	}
}
