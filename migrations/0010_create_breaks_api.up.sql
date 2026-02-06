create table if not exists breaks_api (
                                          id bigserial primary key,

                                          branch_id text not null,
                                          break_id  text not null,
                                          version   bigint not null,

                                          break_date date not null,
                                          start_time text not null,
                                          end_time   text not null,

                                          staff_id   text not null,
                                          room_id    text null,
                                          machine_id text null,
                                          label      text null,
                                          paid_break boolean not null default false,

                                          created_at timestamptz not null default now(),
                                          updated_at timestamptz not null default now(),

                                          unique (branch_id, break_id)
);

create index if not exists idx_breaks_api_branch_date on breaks_api(branch_id, break_date);
create index if not exists idx_breaks_api_staff_date  on breaks_api(staff_id, break_date);