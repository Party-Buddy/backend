package utils

import (
	"party-buddy/internal/schemas/ws"
	"time"
)

func GenBaseMessage(kind *ws.MessageKind) ws.BaseMessage {
	now := time.Now()
	return ws.BaseMessage{Kind: kind, Time: (*ws.Time)(&now)}
}

func GenMessageError(refID *ws.MessageId, code ws.ErrorKind, msg string) ws.MessageError {
	return ws.MessageError{
		BaseMessage: GenBaseMessage(&ws.MsgKindError),
		Error: ws.Error{
			RefId:   refID,
			Code:    code,
			Message: msg,
		},
	}
}
