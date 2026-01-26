CREATE TABLE IF NOT EXISTS public.stock_virtual_transfer_exceptions
(
    transaction_item_id TEXT                                   NOT NULL PRIMARY KEY,
    created_at          TIMESTAMPTZ              DEFAULT now() NOT NULL,
    reason              TEXT                                   NOT NULL,
    purchased_at        TIMESTAMPTZ,
    product_barcode     TEXT                                   NOT NULL,
    product_name        TEXT,
    staff_id            TEXT                                   NOT NULL,
    staff_first_name    TEXT,
    staff_last_name     TEXT
);

CREATE INDEX IF NOT EXISTS idx_svte_created_at
    ON public.stock_virtual_transfer_exceptions (created_at);

CREATE INDEX IF NOT EXISTS idx_svte_reason
    ON public.stock_virtual_transfer_exceptions (reason);

CREATE INDEX IF NOT EXISTS idx_svte_staff_id
    ON public.stock_virtual_transfer_exceptions (staff_id);

CREATE INDEX IF NOT EXISTS idx_svte_product_barcode
    ON public.stock_virtual_transfer_exceptions (product_barcode);

CREATE TABLE IF NOT EXISTS public.stock_virtual_transfers
(
    transaction_item_id TEXT                                   NOT NULL PRIMARY KEY,
    processed_at        TIMESTAMPTZ              DEFAULT now() NOT NULL,
    from_branch_id      TEXT                                   NOT NULL,
    to_branch_id        TEXT                                   NOT NULL,
    barcode             TEXT                                   NOT NULL,
    quantity            INTEGER                                NOT NULL
);