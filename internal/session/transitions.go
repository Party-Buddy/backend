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
	// TODO
	return nil
}

func (u *sessionUpdater) makePollTaskEndedState(s *UnsafeStorage, state *PollStartedState) *TaskEndedState {
	// TODO
	return nil
}
