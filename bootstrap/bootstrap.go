package bootstrap

import (
	"benchmark/cmd/models"
	mongoseeder "benchmark/mongo"
	pgseeder "benchmark/postgres"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type SeedReport struct {
	FixturePath string
	Postgres    struct {
		Categories int64
		Products   int64
	}
	Mongo struct {
		Categories int64
		Products   int64
	}
}

type SeedOptions struct {
	ForceReset bool
}

func Migrate(ctx context.Context, pgConn *pgx.Conn, mongoDB *mongo.Database) error {
	pgSeeder := &pgseeder.Seeder{Conn: pgConn}
	if err := pgSeeder.CreateSchema(ctx, pgConn); err != nil {
		return err
	}

	mongoSeeder := &mongoseeder.Seeder{DB: mongoDB}
	return mongoSeeder.CreateSchema(ctx, mongoDB)
}

func Seed(ctx context.Context, fixture models.Fixture, pgConn *pgx.Conn, mongoDB *mongo.Database, options SeedOptions) (SeedReport, error) {
	pgSeeder := &pgseeder.Seeder{Conn: pgConn}
	mongoSeeder := &mongoseeder.Seeder{DB: mongoDB}

	if err := pgSeeder.Seed(ctx, fixture, options.ForceReset); err != nil {
		return SeedReport{}, err
	}

	if err := mongoSeeder.Seed(ctx, fixture, options.ForceReset); err != nil {
		return SeedReport{}, err
	}

	pgCategoriesCount, pgProductsCount, err := pgSeeder.Count(ctx)
	if err != nil {
		return SeedReport{}, err
	}

	mongoCategoriesCount, mongoProductsCount, err := mongoSeeder.Count(ctx)
	if err != nil {
		return SeedReport{}, err
	}

	if pgCategoriesCount != mongoCategoriesCount || pgProductsCount != mongoProductsCount {
		return SeedReport{}, fmt.Errorf(
			"dataset mismatch: postgres categories=%d products=%d, mongo categories=%d products=%d",
			pgCategoriesCount,
			pgProductsCount,
			mongoCategoriesCount,
			mongoProductsCount,
		)
	}

	report := SeedReport{}
	report.Postgres.Categories = pgCategoriesCount
	report.Postgres.Products = pgProductsCount
	report.Mongo.Categories = mongoCategoriesCount
	report.Mongo.Products = mongoProductsCount

	return report, nil
}

func MigrateAndSeed(ctx context.Context, fixturePath string) (SeedReport, error) {
	return MigrateAndSeedWithOptions(ctx, fixturePath, SeedOptions{})
}

func MigrateAndSeedWithOptions(ctx context.Context, fixturePath string, options SeedOptions) (SeedReport, error) {
	fixture, err := loadFixture(resolveFixturePath(fixturePath))
	if err != nil {
		return SeedReport{}, err
	}

	pgConn, err := connectPostgres(ctx)
	if err != nil {
		return SeedReport{}, err
	}
	defer pgConn.Close(ctx)

	mongoClient, err := connectMongo(ctx)
	if err != nil {
		return SeedReport{}, err
	}
	defer mongoClient.Disconnect(ctx)

	mongoDB := mongoClient.Database("benchmark")

	if err := Migrate(ctx, pgConn, mongoDB); err != nil {
		return SeedReport{}, err
	}

	report, err := Seed(ctx, fixture, pgConn, mongoDB, options)
	if err != nil {
		return SeedReport{}, err
	}

	report.FixturePath = fixturePath
	return report, nil
}

func resolveFixturePath(path string) string {
	if _, err := os.Stat(path); err == nil {
		return path
	}
	candidate := filepath.Join("cmd", "benchmark", path)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return path
}

func loadFixture(path string) (models.Fixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return models.Fixture{}, err
	}

	var fixture models.Fixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return models.Fixture{}, err
	}

	return fixture, nil
}

func connectPostgres(ctx context.Context) (*pgx.Conn, error) {
	return pgx.Connect(ctx, "postgres://postgres:postgres@localhost:5432/benchmark")
}

func connectMongo(ctx context.Context) (*mongo.Client, error) {
	return mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
}
