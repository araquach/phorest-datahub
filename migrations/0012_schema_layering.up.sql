-- Create schemas
CREATE SCHEMA IF NOT EXISTS raw;
CREATE SCHEMA IF NOT EXISTS core;
CREATE SCHEMA IF NOT EXISTS archive;
CREATE SCHEMA IF NOT EXISTS analytics;

-- ------------------------
-- Move raw ingestion tables
-- ------------------------
ALTER TABLE IF EXISTS public.appointments_api SET SCHEMA raw;
ALTER TABLE IF EXISTS public.transactions SET SCHEMA raw;
ALTER TABLE IF EXISTS public.transaction_items SET SCHEMA raw;
ALTER TABLE IF EXISTS public.staff SET SCHEMA raw;
ALTER TABLE IF EXISTS public.staff_worktimetable_slots SET SCHEMA raw;
ALTER TABLE IF EXISTS public.breaks_api SET SCHEMA raw;
ALTER TABLE IF EXISTS public.clients_api SET SCHEMA raw;
ALTER TABLE IF EXISTS public.ph_products SET SCHEMA raw;
ALTER TABLE IF EXISTS public.ph_product_stock SET SCHEMA raw;
ALTER TABLE IF EXISTS public.ph_product_stock_history SET SCHEMA raw;
ALTER TABLE IF EXISTS public.reviews SET SCHEMA raw;
ALTER TABLE IF EXISTS public.branches SET SCHEMA raw;

-- ------------------------
-- Move core business tables
-- ------------------------
ALTER TABLE IF EXISTS public.stock_virtual_transfers SET SCHEMA core;
ALTER TABLE IF EXISTS public.stock_virtual_transfer_exceptions SET SCHEMA core;
ALTER TABLE IF EXISTS public.staff_physical_branch_overrides SET SCHEMA core;
ALTER TABLE IF EXISTS public.staff_person_branch_map_alltime SET SCHEMA core;

-- ------------------------
-- Archive legacy
-- ------------------------
ALTER TABLE IF EXISTS public.clients SET SCHEMA archive;