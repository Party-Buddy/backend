package converters

import (
	"errors"
	"party-buddy/internal/configuration"
	"party-buddy/internal/schemas"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	ws2 "party-buddy/internal/ws"
	"time"
)

func ToPollDuration(pd session.PollDurationer) schemas.DurationType {
	switch t := pd.(type) {
	case session.FixedPollDuration:
		return schemas.DurationType{Kind: schemas.Fixed, Secs: uint16(time.Duration(t).Seconds())}

	case session.DynamicPollDuration:
		return schemas.DurationType{Kind: schemas.Dynamic, Secs: uint16(time.Duration(t).Seconds())}
	}
	panic(errors.New("bad poll duration from server"))
}

func ToSchemaTask(t session.Task) schemas.SchemaTask {
	task := schemas.BaseTask{}
	task.Name = t.Name()
	task.Description = t.Description()
	task.ImgURI = configuration.GenImgURI(t.ImageId().UUID)
	task.Duration = schemas.DurationType{Kind: schemas.Fixed, Secs: uint16(t.TaskDuration().Seconds())}
	switch t := t.(type) {
	case *session.PhotoTask:
		{
			task.Type = schemas.Photo
			task.PollDuration = ToPollDuration(t.PollDuration)
			return &task
		}
	case *session.TextTask:
		{
			task.Type = schemas.Text
			task.PollDuration = ToPollDuration(t.PollDuration)
			return &task
		}
	case *session.CheckedTextTask:
		{
			task.Type = schemas.CheckedText
			return &task
		}
	case *session.ChoiceTask:
		{
			task.Type = schemas.Choice
			return &task
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
	tasks := make([]schemas.SchemaTask, len(g.Tasks))
	for i := 0; i < len(g.Tasks); i++ {
		tasks = append(tasks, ToSchemaTask(g.Tasks[i]))
	}
	game.Tasks = tasks
	game.ImgURI = configuration.GenImgURI(g.ImageId.UUID)
	return game
}

func ToMessageJoined(m session.MsgJoined) ws.MessageJoined {
	msg := ws.MessageJoined{}
	msg.BaseMessage = ws2.genBaseMessage(&ws.MsgKindJoined)
	msg.Sid = m.SessionId.UUID()
	msg.PlayerID = m.PlayerId.UUID().ID()
	msg.Game = ToGameDetails(*m.Game)
	return msg
}
