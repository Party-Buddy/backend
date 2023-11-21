package ws

import (
	"errors"
	"party-buddy/internal/schemas"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
)

func SessionTask2SchemaTask(t session.Task) (schemas.SchemaTask, error) {
	task := schemas.BaseTask{}
	task.Name = t.Name()
	task.Description = t.Description()
	task.ImgURI = t.ImageId().String()
	task.Duration = schemas.DurationType{Kind: schemas.Fixed, Secs: uint16(t.TaskDuration().Seconds())}
	switch t.(type) {
	case *session.PhotoTask:
		{
			sessionPollTask := t.(*session.PhotoTask)
			task.Type = schemas.Photo
			pollTask := schemas.PollTask{}
			pollTask.BaseTask = task

			// TODO: how to set poll duration?
			_ = sessionPollTask
			return &pollTask, nil
		}
	case *session.TextTask:
		{
			sessionPollTask := t.(*session.PhotoTask)
			task.Type = schemas.Text
			pollTask := schemas.PollTask{}
			pollTask.BaseTask = task

			// TODO: how to set poll duration?
			_ = sessionPollTask
			return &pollTask, nil
		}
	case *session.CheckedTextTask:
		{
			task.Type = schemas.CheckedText
			return &task, nil
		}
	case *session.ChoiceTask:
		{
			task.Type = schemas.Choice
			return &task, nil
		}
	default:
		return &schemas.BaseTask{}, errors.New("bad task from server")
	}
}

func SessionGame2GameDetails(g session.Game) (schemas.GameDetails, error) {
	game := schemas.GameDetails{}
	game.Name = g.Name
	game.Description = g.Description
	game.DateChanged = g.DateChanged
	tasks := make([]schemas.SchemaTask, 0)
	for i := 0; i < len(g.Tasks); i++ {
		t, err := SessionTask2SchemaTask(g.Tasks[i])
		if err != nil {
			// TODO: continue or return?
			continue
		}
		tasks = append(tasks, t)
	}
	game.Tasks = tasks
	game.ImgURI = g.ImageId.String() // TODO: convert to URI?
	return game, nil
}

func MsgJoined2MessageJoined(m session.MsgJoined) (ws.MessageJoined, error) {
	msg := ws.MessageJoined{}
	msg.Kind = &ws.MsgKindJoined
	msg.Sid = m.SessionId.UUID()
	msg.PlayerID = m.PlayerId.UUID().ID()
	details, err := SessionGame2GameDetails(*m.Game)
	if err != nil {
		return ws.MessageJoined{}, err
	}
	msg.Game = details
	return msg, nil
}
