CREATE TABLE IF NOT EXISTS chains (
    id BIGSERIAL PRIMARY KEY,
    chain_key TEXT NOT NULL UNIQUE,
    chain_type TEXT NOT NULL,
    name TEXT NOT NULL,
    coin TEXT NOT NULL,
    chain_id BIGINT NOT NULL,
    mainnet BOOLEAN NOT NULL DEFAULT true,
    raw_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS chains_chain_type_idx ON chains (chain_type);
CREATE INDEX IF NOT EXISTS chains_mainnet_idx ON chains (mainnet);
CREATE INDEX IF NOT EXISTS chains_chain_id_idx ON chains (chain_id);
