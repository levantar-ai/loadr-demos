package store

import (
	"context"
	"fmt"
)

const schema = `
CREATE TABLE IF NOT EXISTS products (
	id          BIGSERIAL PRIMARY KEY,
	sku         TEXT UNIQUE NOT NULL,
	name        TEXT NOT NULL,
	price_cents BIGINT NOT NULL,
	stock       INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS orders (
	id          BIGSERIAL PRIMARY KEY,
	customer    TEXT NOT NULL,
	total_cents BIGINT NOT NULL,
	status      TEXT NOT NULL DEFAULT 'pending',
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_items (
	id          BIGSERIAL PRIMARY KEY,
	order_id    BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
	sku         TEXT NOT NULL,
	qty         INT NOT NULL,
	price_cents BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_products_name ON products (name);
`

// Migrate creates the schema if it does not already exist.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

// Seed loads a deterministic catalog the first time the app starts (idempotent:
// it does nothing if products already exist). Generates enough rows that paged
// and search queries are meaningful under load.
func (s *Store) Seed(ctx context.Context) error {
	var count int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM products`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	adjectives := []string{"Rugged", "Compact", "Wireless", "Eco", "Pro", "Mini", "Ultra", "Smart"}
	nouns := []string{"Widget", "Gadget", "Sprocket", "Gizmo", "Doohickey", "Contraption"}

	batch := make([][]any, 0, 200)
	id := 0
	for _, a := range adjectives {
		for _, n := range nouns {
			for v := 1; v <= 4; v++ {
				id++
				sku := fmt.Sprintf("SKU-%04d", id)
				name := fmt.Sprintf("%s %s v%d", a, n, v)
				price := int64(499 + id*37%9000) // pseudo-varied prices
				stock := 1000 + id%500
				batch = append(batch, []any{sku, name, price, stock})
			}
		}
	}

	for _, row := range batch {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO products (sku, name, price_cents, stock)
			VALUES ($1, $2, $3, $4) ON CONFLICT (sku) DO NOTHING`, row...); err != nil {
			return fmt.Errorf("seed: %w", err)
		}
	}
	return nil
}
