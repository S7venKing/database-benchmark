package main

import (
	"benchmark/models"
	mongoseeder "benchmark/mongo"
	pgseeder "benchmark/postgres"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	ctx := context.Background()

	// Load fixture
	data, err := os.ReadFile("fixture/fixture.json")
	if err != nil {
		panic(err)
	}

	var fixture models.Fixture

	err = json.Unmarshal(data, &fixture)
	if err != nil {
		panic(err)
	}

	// PostgreSQL
	pgConn, err := pgx.Connect(
		ctx,
		"postgres://postgres:postgres@localhost:5432/benchmark",
	)
	if err != nil {
		panic(err)
	}
	defer pgConn.Close(ctx)

	pgSeeder := pgseeder.Seeder{
		Conn: pgConn,
	}

	err = pgSeeder.CreateSchema(
		ctx,
		pgConn,
	)
	if err != nil {
		panic(err)
	}

	if err := pgSeeder.Seed(ctx, fixture); err != nil {
		panic(err)
	}

	fmt.Println("PostgreSQL seeded successfully")

	// MongoDB
	mongoClient, err := mongo.Connect(
		options.Client().
			ApplyURI("mongodb://localhost:27017"),
	)
	if err != nil {
		panic(err)
	}
	defer mongoClient.Disconnect(ctx)

	mongoDB := mongoClient.Database("benchmark")

	mongoSeeder := mongoseeder.Seeder{
		DB: mongoDB,
	}

	if err := mongoSeeder.Seed(ctx, fixture); err != nil {
		panic(err)
	}
	// MongoDB
	err = mongoSeeder.CreateSchema(
		ctx,
		mongoDB,
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("MongoDB seeded successfully")

	fmt.Printf(
		"Seed completed: %d categories, %d products\n",
		len(fixture.Categories),
		len(fixture.Products),
	)
}
