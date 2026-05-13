package chains

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

type seedFile struct {
	Chains []json.RawMessage `json:"chains"`
}

type chainMetadata struct {
	Key       string `json:"key"`
	ChainType string `json:"chainType"`
	Name      string `json:"name"`
	Coin      string `json:"coin"`
	ID        int64  `json:"id"`
	Mainnet   bool   `json:"mainnet"`
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ImportFromFile(ctx context.Context, path string) (int, error) {
	if path == "" {
		return 0, nil
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}

	var seed seedFile
	if err := json.Unmarshal(payload, &seed); err != nil {
		return 0, err
	}

	imported := 0
	for index, raw := range seed.Chains {
		var meta chainMetadata
		if err := json.Unmarshal(raw, &meta); err != nil {
			return imported, err
		}
		if meta.Key == "" {
			return imported, fmt.Errorf("chain key is required")
		}

		if err := r.upsert(ctx, meta, index, raw); err != nil {
			return imported, err
		}
		imported++
	}

	return imported, nil
}

func (r *Repository) List(ctx context.Context, page int, size int, key string) ([]json.RawMessage, int, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 30
	}
	if size > 100 {
		size = 100
	}
	keys := parseKeys(key)

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT count(*)
		FROM chains
		WHERE (cardinality($1::text[]) = 0 OR lower(chain_key) = ANY($1::text[]))
	`, keys).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	rows, err := r.db.Query(ctx, `
		SELECT raw_data
		FROM chains
		WHERE (cardinality($1::text[]) = 0 OR lower(chain_key) = ANY($1::text[]))
		ORDER BY sort_order ASC, id ASC
		LIMIT $2 OFFSET $3
	`, keys, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (json.RawMessage, error) {
		var raw json.RawMessage
		if err := row.Scan(&raw); err != nil {
			return nil, err
		}
		return raw, nil
	})
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func parseKeys(value string) []string {
	parts := strings.Split(value, ",")
	keys := make([]string, 0, len(parts))
	for _, part := range parts {
		key := strings.ToLower(strings.TrimSpace(part))
		if key == "" {
			continue
		}
		keys = append(keys, key)
	}

	return keys
}

func (r *Repository) upsert(ctx context.Context, meta chainMetadata, sortOrder int, raw json.RawMessage) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO chains (
			chain_key,
			chain_type,
			name,
			coin,
			chain_id,
			mainnet,
			sort_order,
			raw_data,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		ON CONFLICT (chain_key) DO UPDATE SET
			chain_type = EXCLUDED.chain_type,
			name = EXCLUDED.name,
			coin = EXCLUDED.coin,
			chain_id = EXCLUDED.chain_id,
			mainnet = EXCLUDED.mainnet,
			sort_order = EXCLUDED.sort_order,
			raw_data = EXCLUDED.raw_data,
			updated_at = now()
	`, meta.Key, meta.ChainType, meta.Name, meta.Coin, meta.ID, meta.Mainnet, sortOrder, raw)

	return err
}
