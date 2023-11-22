BEGIN;

INSERT INTO users (id, role)
    VALUES
        ('deadbeef-1337-1337-1337-1712abad1dea', 'admin'),
        ('c0de900d-1234-1234-1234-addc0ffee700', 'admin');

INSERT INTO games (id, name, owner_id, description, image_id)
    VALUES
        (
            '11112222-3333-4444-5555-131072262144', -- id
            'Test game', -- name
            'deadbeef-1337-1337-1337-1712abad1dea', -- owner_id
            'pls ignore.', -- description
            NULL -- image_id
        ),
        (
            '66667777-8888-9999-0000-167772161024', -- id
            'Another game', -- name
            'c0de900d-1234-1234-1234-addc0ffee700', -- owner_id
            'ignore this too plskthx', -- description
            NULL -- image_id
        );

INSERT INTO tasks (id, name, owner_id, description, image_id, duration_secs, poll_duration_secs, poll_duration_type, task_kind)
    VALUES
        (
            '12345678-1234-1234-1234-123456789abc', -- id
            '65 лет', -- name
            'deadbeef-1337-1337-1337-1712abad1dea', -- owner_id
            'Какой из этих вариантов — самая частая причина самоубийств среди программистов?', -- description
            NULL, -- image_id
            30, -- duration_secs
            0, -- poll_duration_secs
            'fixed', -- poll_duration_type
            'choice' -- task_kind
        ),
        (
            '11111111-2222-3333-4444-555555555555', -- id
            'Защита без опасности', -- name
            'c0de900d-1234-1234-1234-addc0ffee700', -- owner_id
            'Какое насекомое шифруется?', -- description
            NULL,
            60, -- duration_secs
            0, -- poll_duration_secs
            'fixed', -- poll_duration_type
            'checked-text' -- task_kind
        ),
        (
            '12121212-3333-4444-5555-678678678678', -- id
            'Наногуманизм', -- name
            'deadbeef-1337-1337-1337-1712abad1dea', -- owner_id
            'Кого способна отследить статистика?', -- description
            NULL,
            15, -- duration_secs
            0, -- poll_duration_secs
            'fixed', -- poll_duration_type
            'choice' -- task_kind
        ),
        (
            '12481632-1024-2048-4096-819221483648', -- id
            'Текстбокс фортуны', -- name
            'c0de900d-1234-1234-1234-addc0ffee700', -- owner_id
            'Какой ответ выиграет?', -- description
            NULL,
            45, -- duration_secs
            15, -- poll_duration_secs
            'dynamic', -- poll_duration_type
            'text' -- task_kind
        ),
        (
            'c001d05e-ba17-7e57-da7a-57ab1eca7ba7', -- id
            'Специальная теория относительности', -- name
            'deadbeef-1337-1337-1337-1712abad1dea', -- owner_id
            'Делу — время. А что потехе?', -- description
            NULL,
            60, -- duration_secs
            60, -- poll_duration_secs
            'fixed', -- poll_duration_type
            'text' -- task_kind
        );

INSERT INTO checked_text_tasks (task_id, answer)
    VALUES
        ('11111111-2222-3333-4444-555555555555', 'КУЗНЕЧИК');

INSERT INTO choice_task_options (task_id, alternative, correct)
    VALUES
        ('12345678-1234-1234-1234-123456789abc', 'Миграция БД', false),
        ('12345678-1234-1234-1234-123456789abc', 'Бурение легаси', false),
        ('12345678-1234-1234-1234-123456789abc', 'Docker', true),
        ('12345678-1234-1234-1234-123456789abc', 'Выгоревшие дедлайны', false),

        ('12121212-3333-4444-5555-678678678678', 'микрочелики', false),
        ('12121212-3333-4444-5555-678678678678', 'макрочелики', false),
        ('12121212-3333-4444-5555-678678678678', 'милличелики', true),
        ('12121212-3333-4444-5555-678678678678', 'миничелики', false);

INSERT INTO game_tasks (game_id, task_idx, task_id)
    VALUES
        ('11112222-3333-4444-5555-131072262144', 0, '12345678-1234-1234-1234-123456789abc'),
        ('11112222-3333-4444-5555-131072262144', 1, '12121212-3333-4444-5555-678678678678'),
        ('11112222-3333-4444-5555-131072262144', 2, '12481632-1024-2048-4096-819221483648'),

        ('66667777-8888-9999-0000-167772161024', 0, '11111111-2222-3333-4444-555555555555'),
        ('66667777-8888-9999-0000-167772161024', 1, 'c001d05e-ba17-7e57-da7a-57ab1eca7ba7');

COMMIT;
