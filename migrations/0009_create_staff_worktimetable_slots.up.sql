CREATE TABLE IF NOT EXISTS staff_worktimetable_slots (
                                                         id BIGSERIAL PRIMARY KEY,

                                                         branch_id TEXT NOT NULL,
                                                         staff_id  TEXT NOT NULL,

                                                         slot_date  DATE NOT NULL,
                                                         start_time time NOT NULL,
                                                         end_time   time NOT NULL,

                                                         time_off_start_time time NULL,
                                                         time_off_end_time   time NULL,

                                                         type TEXT NOT NULL,
                                                         custom TEXT NULL,

                                                         slot_branch_id TEXT NULL,
                                                         work_activity_id TEXT NULL,

                                                         created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                                         updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Prevent duplicates (idempotent inserts)
CREATE UNIQUE INDEX IF NOT EXISTS ux_worktimetable_slot
    ON staff_worktimetable_slots (
                                  branch_id, staff_id, slot_date, start_time, end_time, type,
                                  COALESCE(work_activity_id, ''),
                                  COALESCE(custom, '')
        );

CREATE INDEX IF NOT EXISTS idx_worktimetable_staff_date
    ON staff_worktimetable_slots (staff_id, slot_date);

CREATE INDEX IF NOT EXISTS idx_worktimetable_branch_date
    ON staff_worktimetable_slots (branch_id, slot_date);