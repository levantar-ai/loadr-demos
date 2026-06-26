// Package store is the Postgres data layer for the demo storefront.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// Store wraps a pgx connection pool.
type Store struct{ pool *pgxpool.Pool }

// Product is a catalog item.
type Product struct {
	ID         int64  `json:"id"`
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"`
	Stock      int    `json:"stock"`
}

// OrderItem is a single line in an order request.
type OrderItem struct {
	SKU string `json:"sku"`
	Qty int    `json:"qty"`
}

// Order is a placed order.
type Order struct {
	ID         int64       `json:"id"`
	Customer   string      `json:"customer"`
	TotalCents int64       `json:"total_cents"`
	Status     string      `json:"status"`
	Items      []OrderItem `json:"items"`
	CreatedAt  time.Time   `json:"created_at"`
}

// New connects to Postgres, retrying for up to 30s so it tolerates a database
// container that is still starting up (common in CI service containers).
func New(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = 20

	deadline := time.Now().Add(30 * time.Second)
	for {
		pool, perr := pgxpool.NewWithConfig(ctx, cfg)
		if perr == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				return &Store{pool: pool}, nil
			} else {
				pool.Close()
				perr = pingErr
			}
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("connect: %w", perr)
		}
		time.Sleep(time.Second)
	}
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// Ping checks database connectivity (used by the readiness probe).
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// ListProducts returns products, optionally filtered by a case-insensitive
// name search, with limit/offset paging.
func (s *Store) ListProducts(ctx context.Context, q string, limit, offset int) ([]Product, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, sku, name, price_cents, stock
		FROM products
		WHERE ($1 = '' OR name ILIKE '%' || $1 || '%')
		ORDER BY id
		LIMIT $2 OFFSET $3`, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Product, 0, limit)
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.PriceCents, &p.Stock); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetProduct fetches a single product by id.
func (s *Store) GetProduct(ctx context.Context, id int64) (*Product, error) {
	var p Product
	err := s.pool.QueryRow(ctx, `
		SELECT id, sku, name, price_cents, stock FROM products WHERE id = $1`, id).
		Scan(&p.ID, &p.SKU, &p.Name, &p.PriceCents, &p.Stock)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// CreateProduct inserts a product and returns its id.
func (s *Store) CreateProduct(ctx context.Context, p Product) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO products (sku, name, price_cents, stock)
		VALUES ($1, $2, $3, $4)
		RETURNING id`, p.SKU, p.Name, p.PriceCents, p.Stock).Scan(&id)
	return id, err
}

// CreateOrder places an order in a single transaction: it prices each line
// from the catalog, decrements stock, and writes the order + items.
func (s *Store) CreateOrder(ctx context.Context, customer string, items []OrderItem) (*Order, error) {
	if len(items) == 0 {
		return nil, errors.New("order has no items")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	var total int64
	for _, it := range items {
		if it.Qty <= 0 {
			return nil, fmt.Errorf("invalid qty for %s", it.SKU)
		}
		var price int64
		err := tx.QueryRow(ctx, `
			UPDATE products SET stock = stock - $1
			WHERE sku = $2 AND stock >= $1
			RETURNING price_cents`, it.Qty, it.SKU).Scan(&price)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: sku %s unavailable", ErrNotFound, it.SKU)
		}
		if err != nil {
			return nil, err
		}
		total += price * int64(it.Qty)
	}

	o := &Order{Customer: customer, TotalCents: total, Status: "confirmed", Items: items}
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (customer, total_cents, status)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`, o.Customer, o.TotalCents, o.Status).Scan(&o.ID, &o.CreatedAt)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_items (order_id, sku, qty, price_cents)
			SELECT $1, $2, $3, price_cents FROM products WHERE sku = $2`,
			o.ID, it.SKU, it.Qty); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return o, nil
}

// GetOrder fetches an order and its line items.
func (s *Store) GetOrder(ctx context.Context, id int64) (*Order, error) {
	var o Order
	err := s.pool.QueryRow(ctx, `
		SELECT id, customer, total_cents, status, created_at FROM orders WHERE id = $1`, id).
		Scan(&o.ID, &o.Customer, &o.TotalCents, &o.Status, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `SELECT sku, qty FROM order_items WHERE order_id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.SKU, &it.Qty); err != nil {
			return nil, err
		}
		o.Items = append(o.Items, it)
	}
	return &o, rows.Err()
}
