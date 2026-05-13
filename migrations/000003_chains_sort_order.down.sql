DROP INDEX IF EXISTS chains_sort_order_idx;

ALTER TABLE chains
DROP COLUMN IF EXISTS sort_order;
