CREATE TABLE IF NOT EXISTS public.appointments_api (
                                                       id BIGSERIAL PRIMARY KEY,

                                                       branch_id      TEXT NOT NULL,
                                                       appointment_id TEXT NOT NULL,

                                                       version BIGINT NOT NULL DEFAULT 0,

                                                       appointment_date DATE NOT NULL,
                                                       start_time TIME NOT NULL,
                                                       end_time   TIME NOT NULL,

                                                       price NUMERIC(12,2) NOT NULL DEFAULT 0,

                                                       staff_id TEXT NOT NULL,
                                                       confirmed BOOLEAN NOT NULL DEFAULT FALSE,
                                                       service_id TEXT NOT NULL,

                                                       created_at_phorest TIMESTAMPTZ,
                                                       updated_at_phorest TIMESTAMPTZ,

                                                       staff_request   BOOLEAN NOT NULL DEFAULT FALSE,
                                                       preferred_staff BOOLEAN NOT NULL DEFAULT FALSE,
                                                       client_id       TEXT NOT NULL,

                                                       service_reward_id    TEXT,
                                                       purchasing_branch_id TEXT,
                                                       service_name         TEXT,

                                                       state            TEXT NOT NULL,
                                                       activation_state TEXT NOT NULL,

                                                       deposit_amount   NUMERIC(12,2),
                                                       deposit_datetime TIMESTAMPTZ,

                                                       booking_id TEXT,
                                                       source     TEXT,

                                                       deleted BOOLEAN NOT NULL DEFAULT FALSE,

                                                       internet_service_categories JSONB,

                                                       created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                                       updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

                                                       CONSTRAINT uq_appointments_api_branch_appt UNIQUE (branch_id, appointment_id)
);

-- Indexes for incremental sync and KPIs
CREATE INDEX IF NOT EXISTS idx_appointments_api_branch_updated
    ON public.appointments_api (branch_id, updated_at_phorest);

CREATE INDEX IF NOT EXISTS idx_appointments_api_branch_date
    ON public.appointments_api (branch_id, appointment_date);

CREATE INDEX IF NOT EXISTS idx_appointments_api_branch_staff_date
    ON public.appointments_api (branch_id, staff_id, appointment_date);

CREATE INDEX IF NOT EXISTS idx_appointments_api_branch_client_date
    ON public.appointments_api (branch_id, client_id, appointment_date);

CREATE INDEX IF NOT EXISTS idx_appointments_api_branch_service_date
    ON public.appointments_api (branch_id, service_id, appointment_date);

CREATE INDEX IF NOT EXISTS idx_appointments_api_state
    ON public.appointments_api (state);