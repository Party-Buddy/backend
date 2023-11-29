package converters

import (
	"party-buddy/internal/configuration"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/ws/utils"
)

func ToMessageTaskEnd(m session.MsgTaskEnd) ws.MessageTaskEnd {
	msg := ws.MessageTaskEnd{
		BaseMessage: utils.GenBaseMessage(&ws.MsgKindTaskEnd),
		TaskIdx:     uint8(m.TaskIdx),
		Deadline:    m.Deadline,
	}
	answers := make([]ws.Answer, 0)
	for i := 0; i < len(m.Answers); i++ {
		switch v := m.Answers[i].Value.(type) {
		case *session.ChoiceEventAnswer:
			answers = append(answers, &ws.TaskOptionAnswer{
				Value:       string(*v),
				PlayerCount: m.Answers[i].PlayerCount,
				Correct:     m.Answers[i].Correct,
			})

		case *session.CheckedTextEventAnswer:
			answers = append(answers, &ws.CheckedWordAnswer{
				Value:       string(*v),
				PlayerCount: m.Answers[i].PlayerCount,
				Correct:     m.Answers[i].Correct,
			})

		case *session.PhotoEventAnswer:
			answers = append(answers, &ws.PhotoAnswer{
				Value: configuration.GenImgURI(session.ImageID(*v).UUID),
				Votes: m.Answers[i].PlayerCount,
			})

		case *session.TextEventAnswer:
			answers = append(answers, &ws.PhotoAnswer{
				Value: string(*v),
				Votes: m.Answers[i].PlayerCount,
			})
		}
	}
	msg.Answers = answers
	// TODO: convert scoreboard
	return msg
}
