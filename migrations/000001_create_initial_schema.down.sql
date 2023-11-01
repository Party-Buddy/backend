BEGIN;

DROP VIEW image_refs_view;
DROP INDEX task_image_id_idx;
DROP INDEX game_image_id_idx;
DROP TABLE choice_task_options;
DROP TABLE checked_text_tasks;
DROP TABLE tasks;
DROP TABLE games;
DROP TABLE users;
DROP TABLE session_image_refs;
DROP TABLE images;

COMMIT;
