-- Revert TIME -> text (canonical HH:MM:SS)

ALTER TABLE breaks_api
    ALTER COLUMN start_time TYPE text USING to_char(start_time, 'HH24:MI:SS'),
    ALTER COLUMN end_time   TYPE text USING to_char(end_time, 'HH24:MI:SS');