package converters

import (
	"errors"
	"party-buddy/internal/configuration"
	"party-buddy/internal/schemas"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/ws/utils"
	"time"
)

func ToPollDuration(pd session.PollDurationer) schemas.PollDuration {
	switch t := pd.(type) {
	case session.FixedPollDuration:
		return schemas.PollDuration{Kind: schemas.Fixed, Secs: uint16(time.Duration(t).Seconds())}

	case session.DynamicPollDuration:
		return schemas.PollDuration{Kind: schemas.Dynamic, Secs: uint16(time.Duration(t).Seconds())}
	}
	panic(errors.New("bad poll duration from server"))
}

func ToSchemaTask(t session.Task) schemas.BaseTaskWithImg {
	task := schemas.BaseTaskWithImg{}
	task.Name = t.GetName()
	task.Description = t.GetDescription()
	task.ImgURI = configuration.GenImgURI(t.GetImageID().UUID)
	task.Duration = schemas.PollDuration{Kind: schemas.Fixed, Secs: uint16(t.GetTaskDuration().Seconds())}
	switch t := t.(type) {
	case *session.PhotoTask:
		{
			task.Type = schemas.Photo
			task.PollDuration = ToPollDuration(t.PollDuration)
			return task
		}
	case *session.TextTask:
		{
			task.Type = schemas.Text
			task.PollDuration = ToPollDuration(t.PollDuration)
			return task
		}
	case *session.CheckedTextTask:
		{
			task.Type = schemas.CheckedText
			return task
		}
	case *session.ChoiceTask:
		{
			task.Type = schemas.Choice
			return task
		}
	default:
		panic(errors.New("bad task from server"))
	}
}

func ToGameDetails(g session.Game) schemas.GameDetails {
	game := schemas.GameDetails{}
	game.Name = g.Name
	game.Description = g.Description
	game.DateChanged = g.DateChanged
	tasks := make([]schemas.BaseTaskWithImg, len(g.Tasks))
	for i := 0; i < len(g.Tasks); i++ {
		tasks = append(tasks, ToSchemaTask(g.Tasks[i]))
	}
	game.Tasks = tasks
	game.ImgURI = configuration.GenImgURI(g.ImageID.UUID)
	return game
}

func ToMessageJoined(m session.MsgJoined) ws.MessageJoined {
	msg := ws.MessageJoined{}
	msg.BaseMessage = utils.GenBaseMessage(&ws.MsgKindJoined)
	msg.Sid = m.SessionID.UUID()
	msg.PlayerID = uint32(m.PlayerID)
	msg.Game = ToGameDetails(*m.Game)
	return msg
}
