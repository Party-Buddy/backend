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
		Deadline:    ws.Time(m.Deadline),
	}

	for _, score := range m.Scoreboard.Scores() {
		msg.Scoreboard = append(msg.Scoreboard, ws.TaskPlayerScore{
			PlayerID:    uint32(score.PlayerID),
			TaskPoints:  uint32(m.Winners[score.PlayerID]),
			TotalPoints: uint32(score.Score),
		})
	}

	for _, a := range m.Results {
		switch t := m.Task.(type) {
		case session.ChoiceTask:
			msg.Answers = append(msg.Answers, &ws.TaskOptionAnswer{
				Value:       t.Options[a.Value.(session.ChoiceTaskAnswer)],
				PlayerCount: uint16(a.Submissions),
				Correct:     t.AnswerIdx == int(a.Value.(session.ChoiceTaskAnswer)),
			})

		case session.CheckedTextTask:
			msg.Answers = append(msg.Answers, &ws.CheckedWordAnswer{
				Value:       string(a.Value.(session.CheckedTextAnswer)),
				PlayerCount: uint16(a.Submissions),
				Correct:     t.Answer == string(a.Value.(session.CheckedTextAnswer)),
			})

		case session.PhotoTask:
			msg.Answers = append(msg.Answers, &ws.PhotoAnswer{
				Value: configuration.GenImgURI(a.Value.(session.PhotoTaskAnswer).UUID),
				Votes: uint16(a.Votes),
			})

		case session.TextTask:
			msg.Answers = append(msg.Answers, &ws.WordAnswer{
				Value: string(a.Value.(session.TextTaskAnswer)),
				Votes: uint16(a.Votes),
			})
		}
	}

	return msg
}
