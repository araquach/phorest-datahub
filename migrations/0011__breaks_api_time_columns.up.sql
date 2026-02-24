-- Normalise Phorest time strings (e.g. 09:00:00.000) and convert to TIME

-- 1) Strip milliseconds if present
UPDATE breaks_api
SET start_time = regexp_replace(start_time, '\.\d+$', '')
WHERE start_time ~ '\.\d+$';

UPDATE breaks_api
SET end_time = regexp_replace(end_time, '\.\d+$', '')
WHERE end_time ~ '\.\d+$';

-- 2) Convert text -> time
ALTER TABLE breaks_api
    ALTER COLUMN start_time TYPE time USING start_time::time,
    ALTER COLUMN end_time   TYPE time USING end_time::time;