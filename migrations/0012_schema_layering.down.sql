ALTER TABLE IF EXISTS raw.appointments_api SET SCHEMA public;
ALTER TABLE IF EXISTS raw.transactions SET SCHEMA public;
ALTER TABLE IF EXISTS raw.transaction_items SET SCHEMA public;
ALTER TABLE IF EXISTS raw.staff SET SCHEMA public;
ALTER TABLE IF EXISTS raw.staff_worktimetable_slots SET SCHEMA public;
ALTER TABLE IF EXISTS raw.breaks_api SET SCHEMA public;
ALTER TABLE IF EXISTS raw.clients_api SET SCHEMA public;
ALTER TABLE IF EXISTS raw.ph_products SET SCHEMA public;
ALTER TABLE IF EXISTS raw.ph_product_stock SET SCHEMA public;
ALTER TABLE IF EXISTS raw.ph_product_stock_history SET SCHEMA public;
ALTER TABLE IF EXISTS raw.reviews SET SCHEMA public;
ALTER TABLE IF EXISTS raw.branches SET SCHEMA public;

ALTER TABLE IF EXISTS core.stock_virtual_transfers SET SCHEMA public;
ALTER TABLE IF EXISTS core.stock_virtual_transfer_exceptions SET SCHEMA public;
ALTER TABLE IF EXISTS core.staff_physical_branch_overrides SET SCHEMA public;
ALTER TABLE IF EXISTS core.staff_person_branch_map_alltime SET SCHEMA public;

ALTER TABLE IF EXISTS archive.clients SET SCHEMA public;