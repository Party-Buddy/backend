BEGIN;

-- image metadata.
CREATE TABLE images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- booleans don't seem like a good idea.
    -- but anyway: serve the image iff uploaded == true,
    -- and block uploads if read_only == true
    uploaded BOOLEAN NOT NULL DEFAULT FALSE,
    read_only BOOLEAN NOT NULL DEFAULT FALSE,
    owner_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- tracks images referenced in live sessions.
-- this table should be TRUNCATEd on server startup.
-- when a session is created, its image refs should be added here.
-- when a session ends, the server should DELETE rows with its session id.
CREATE TABLE session_image_refs (
    image_id UUID NOT NULL REFERENCES images,
    session_id UUID NOT NULL,

    PRIMARY KEY (image_id, session_id)
);

-- at the moment this table only contains administrators.
-- can be expanded later if so desired.
CREATE TABLE users (
    id UUID PRIMARY KEY,

    -- "admin"
    role TEXT NOT NULL
);

CREATE TABLE games (
    id UUID PRIMARY KEY,

    name TEXT NOT NULL,
    owner_id UUID NOT NULL,
    description TEXT NOT NULL,
    image_id UUID NULL REFERENCES images,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE tasks (
    id UUID PRIMARY KEY,

    name TEXT NOT NULL,
    owner_id UUID NOT NULL,
    description TEXT NOT NULL,
    image_id UUID NULL REFERENCES images,
    duration_secs INTEGER NOT NULL,
    poll_duration_secs INTEGER NOT NULL,
    -- "fixed" | "dynamic"
    poll_duration_type TEXT NOT NULL,
    -- "photo" | "text" | "checked-text" | "choice"
    task_kind TEXT NOT NULL
);

-- additional info for checked-text tasks.
-- 1:1.
CREATE TABLE checked_text_tasks (
    task_id UUID PRIMARY KEY REFERENCES tasks ON DELETE CASCADE,
    answer TEXT NOT NULL
);

-- the options for choice tasks.
-- n:1.
--
-- not named "choice_tasks" to allow for future expansion.
CREATE TABLE choice_task_options (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    task_id UUID REFERENCES tasks ON DELETE CASCADE,

    -- `option` is a context-specific keyword which i'd rather avoid...
    -- so "alternative" it is.
    alternative TEXT NOT NULL,
    correct BOOLEAN NOT NULL
);

-- two indexes for bookkeeping.
CREATE INDEX game_image_id_idx
    ON games (image_id)
    WHERE image_id IS NOT NULL;

CREATE INDEX task_image_id_idx
    ON tasks (image_id)
    WHERE image_id IS NOT NULL;

-- a view for counting image references.
-- (alternately, this could be made SELECT DISTINCT image_id,
-- without the ref_count.)
-- relies on the indexes above for performance
CREATE VIEW image_refs_view AS
    SELECT image_id, COUNT(*) AS ref_count
        FROM (
            SELECT image_id
                FROM session_image_refs
            UNION
            SELECT image_id
                FROM games
                WHERE image_id IS NOT NULL
            UNION
            SELECT image_id
                FROM tasks
                WHERE image_id IS NOT NULL
        ) AS refs
        GROUP BY image_id;

COMMIT;
