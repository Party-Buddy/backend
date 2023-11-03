BEGIN;

-- associates tasks to games.
-- the order of tasks is defined by task_idx.
-- NOTE: indices are not required to be contiguous.
CREATE TABLE game_tasks (
    game_id UUID REFERENCES games,
    task_idx INTEGER,
    task_id UUID NOT NULL REFERENCES tasks,

    PRIMARY KEY (game_id, task_idx)
);

COMMIT;
