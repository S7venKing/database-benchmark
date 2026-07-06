package postgres

import (
	"benchmark/cmd/models"
	"context"

	"github.com/jackc/pgx/v5"
)

type Seeder struct {
	Conn *pgx.Conn
}

func (*Seeder) CreateSchema(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS categories (
			id INT PRIMARY KEY,
			name VARCHAR(100) NOT NULL
		);

		CREATE TABLE IF NOT EXISTS products (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			category_id INT NOT NULL,
			price NUMERIC(12,2) NOT NULL,
			tags TEXT[],
			created_at TIMESTAMP NOT NULL,
			CONSTRAINT fk_products_categories
				FOREIGN KEY (category_id)
				REFERENCES categories(id)
		);

		CREATE INDEX IF NOT EXISTS idx_products_category
			ON products(category_id);

		CREATE INDEX IF NOT EXISTS idx_products_price
			ON products(price);
	`)

	return err
}

func (s *Seeder) Count(ctx context.Context) (int64, int64, error) {
	var categories int64
	if err := s.Conn.QueryRow(ctx, `SELECT COUNT(*) FROM categories`).Scan(&categories); err != nil {
		return 0, 0, err
	}

	var products int64
	if err := s.Conn.QueryRow(ctx, `SELECT COUNT(*) FROM products`).Scan(&products); err != nil {
		return 0, 0, err
	}

	return categories, products, nil
}

func (s *Seeder) Seed(ctx context.Context, fixture models.Fixture) error {

	_, err := s.Conn.Exec(ctx, `
		TRUNCATE TABLE products RESTART IDENTITY CASCADE;
		TRUNCATE TABLE categories RESTART IDENTITY CASCADE;
	`)
	if err != nil {
		return err
	}

	for _, c := range fixture.Categories {
		_, err = s.Conn.Exec(ctx, `
			INSERT INTO categories(id,name)
			VALUES($1,$2)
		`, c.ID, c.Name)

		if err != nil {
			return err
		}
	}

	rows := make([][]any, 0, len(fixture.Products))

	for _, p := range fixture.Products {
		rows = append(rows, []any{
			p.ID,
			p.Name,
			p.CategoryID,
			p.Price,
			p.Tags,
			p.CreatedAt,
		})
	}

	_, err = s.Conn.CopyFrom(
		ctx,
		pgx.Identifier{"products"},
		[]string{
			"id",
			"name",
			"category_id",
			"price",
			"tags",
			"created_at",
		},
		pgx.CopyFromRows(rows),
	)

	return err
}
