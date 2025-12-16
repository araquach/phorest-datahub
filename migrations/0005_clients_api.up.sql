CREATE TABLE clients_api (
                             id                           bigserial PRIMARY KEY,
                             client_id                    text        NOT NULL,
                             version                      bigint,
                             first_name                   text,
                             last_name                    text,
                             mobile                       text,
                             linked_client_mobile         text,
                             land_line                    text,
                             email                        text,

                             street_address_1             text,
                             street_address_2             text,
                             city                         text,
                             state                        text,
                             postal_code                  text,
                             country                      text,

                             birth_date                   date,
                             client_since                 timestamptz,
                             gender                       text,
                             notes                        text,

                             sms_marketing_consent        boolean,
                             email_marketing_consent      boolean,
                             sms_reminder_consent         boolean,
                             email_reminder_consent       boolean,

                             preferred_staff_id           text,
                             external_id                  text,
                             creating_branch_id           text,

                             archived                     boolean,
                             banned                       boolean,
                             merged_to_client_id          text,
                             deleted                      boolean,

                             client_category_ids          text,

                             first_visit                  date,
                             last_visit                   date,

                             created_at_phorest           timestamptz,
                             updated_at_phorest           timestamptz,

                             photo_url                    text,

                             loyalty_card_serial          text,
                             loyalty_points               numeric,

                             credit_outstanding_balance   numeric,
                             credit_days                  bigint,
                             credit_limit                 numeric,

                             created_at                   timestamptz DEFAULT now(),
                             updated_at                   timestamptz DEFAULT now()
);

CREATE UNIQUE INDEX idx_clients_api_client_id
    ON clients_api (client_id);

CREATE INDEX idx_clients_api_updated_at_phorest
    ON clients_api (updated_at_phorest);