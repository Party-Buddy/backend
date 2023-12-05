package session

import "time"

func (u *sessionUpdater) makeGameStartedState(s *UnsafeStorage, state *AwaitingPlayersState) *GameStartedState {
	return &GameStartedState{
		deadline: time.Now().Add(GameStartedTimeout),
	}
}

func (u *sessionUpdater) makeFirstTaskStartedState(s *UnsafeStorage, state *GameStartedState) *TaskStartedState {
	task := s.taskByIdx(u.sid, 0)
	if task == nil {
		u.log.Panicf("task 0 not found")
	}

	return &TaskStartedState{
		taskIdx:  0,
		deadline: time.Now().Add(task.GetTaskDuration()),
		answers:  make(map[PlayerID]TaskAnswer),
		ready:    make(map[PlayerID]struct{}),
	}
}

func (u *sessionUpdater) makeNextTaskStartedState(s *UnsafeStorage, state *TaskEndedState) *TaskStartedState {
	task := s.taskByIdx(u.sid, state.taskIdx+1)
	if task == nil {
		u.log.Panicf("task %d not found", state.taskIdx+1)
	}

	return &TaskStartedState{
		taskIdx:  state.taskIdx + 1,
		deadline: time.Now().Add(task.GetTaskDuration()),
		answers:  make(map[PlayerID]TaskAnswer),
		ready:    make(map[PlayerID]struct{}),
	}
}

func (u *sessionUpdater) makePollStartedState(s *UnsafeStorage, state *TaskStartedState) *PollStartedState {
	// TODO
	return nil
}

func (u *sessionUpdater) makePlainTaskEndedState(s *UnsafeStorage, state *TaskStartedState) *TaskEndedState {
	var results []AnswerResult
	winners := make(map[PlayerID]Score)

	switch task := s.taskByIdx(u.sid, state.taskIdx).(type) {
	case CheckedTextTask:
		answerIndices := make(map[string]int)

		for _, player := range s.Players(u.sid) {
			answerOpaque, ok := state.answers[player.ID]
			if !ok {
				continue
			}

			answer := answerOpaque.(CheckedTextAnswer)
			answerStr := string(answer)

			idx, ok := answerIndices[answerStr]
			if !ok {
				idx = len(results)
				results = append(results, AnswerResult{
					Value: answer,
				})
				answerIndices[answerStr] = idx
			}

			results[idx].Submissions++

			if answerStr == task.Answer {
				winners[player.ID] = CheckedTextTaskPoints
			}
		}

	case ChoiceTask:
		for i := range task.Options {
			results = append(results, AnswerResult{
				Value: ChoiceTaskAnswer(i),
			})
		}

		for _, player := range s.Players(u.sid) {
			answerOpaque, ok := state.answers[player.ID]
			if !ok {
				continue
			}

			answer := answerOpaque.(ChoiceTaskAnswer)
			idx := int(answer)

			results[idx].Submissions++

			if idx == task.AnswerIdx {
				winners[player.ID] = ChoiceTaskPoints
			}
		}

	default:
		u.log.Panicf(
			"cannot make *TaskEndedState from *TaskStartedState: task %d (%T) requires a poll",
			state.taskIdx,
			task,
		)
	}

	return &TaskEndedState{
		taskIdx:  state.taskIdx,
		deadline: time.Now().Add(TaskEndTimeout),
		results:  results,
		winners:  winners,
	}
}

func (u *sessionUpdater) makePollTaskEndedState(s *UnsafeStorage, state *PollStartedState) *TaskEndedState {
	// TODO
	return nil
}
