CREATE TABLE IF NOT EXISTS public.staff_physical_branch_overrides
(
    staff_id           TEXT                                   NOT NULL PRIMARY KEY,
    physical_branch_id TEXT                                   NOT NULL,
    active             BOOLEAN                  DEFAULT TRUE  NOT NULL,
    updated_at         TIMESTAMPTZ              DEFAULT now() NOT NULL
);