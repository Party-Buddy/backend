package utils

import (
	"party-buddy/internal/schemas/ws"
	"time"
)

func GenBaseMessage(kind *ws.MessageKind) ws.BaseMessage {
	now := time.Now()
	return ws.BaseMessage{Kind: kind, Time: (*ws.Time)(&now)}
}

func GenMessageError(refID *ws.MessageID, code ws.ErrorKind, msg string) ws.MessageError {
	return ws.MessageError{
		BaseMessage: GenBaseMessage(&ws.MsgKindError),
		Error: ws.Error{
			RefID:   refID,
			Code:    code,
			Message: msg,
		},
	}
}
