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
	for _, a := range m.AnswerResults {
		switch t := m.Task.(type) {
		case *session.ChoiceTask:
			answers = append(answers, &ws.TaskOptionAnswer{
				Value:       t.Options[a.Value.(session.ChoiceTaskAnswer)],
				PlayerCount: uint16(a.Submissions),
				Correct:     t.AnswerIdx == int(a.Value.(session.ChoiceTaskAnswer)),
			})

		case *session.CheckedTextTask:
			answers = append(answers, &ws.CheckedWordAnswer{
				Value:       string(a.Value.(session.CheckedTextAnswer)),
				PlayerCount: uint16(a.Submissions),
				Correct:     t.Answer == string(a.Value.(session.CheckedTextAnswer)),
			})

		case *session.PhotoTask:
			answers = append(answers, &ws.PhotoAnswer{
				Value: configuration.GenImgURI(a.Value.(session.PhotoTaskAnswer).UUID),
				Votes: uint16(a.Votes),
			})

		case *session.TextTask:
			answers = append(answers, &ws.WordAnswer{
				Value: string(a.Value.(session.TextTaskAnswer)),
				Votes: uint16(a.Votes),
			})
		}
	}
	msg.Answers = answers
	// TODO: convert scoreboard
	return msg
}
