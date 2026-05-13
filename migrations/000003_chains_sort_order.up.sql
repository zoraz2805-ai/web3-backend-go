ALTER TABLE chains
ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS chains_sort_order_idx ON chains (sort_order);
