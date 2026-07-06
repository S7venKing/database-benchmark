package main

import (
	"benchmark/bootstrap"
	"benchmark/cmd/models"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Result struct {
	Shape  string  `json:"shape"`
	Engine string  `json:"engine"`
	TimeMs float64 `json:"timeMs"`
}

type FixtureCheck struct {
	FixtureID  string  `json:"fixtureId"`
	PGName     string  `json:"pgName"`
	MongoName  string  `json:"mongoName"`
	PGPrice    float64 `json:"pgPrice"`
	MongoPrice float64 `json:"mongoPrice"`
	Match      bool    `json:"match"`
}

type postgresQuery struct {
	Name string
	File string
	SQL  string
	Args []any
}

type mongoQuery struct {
	Name    string
	File    string
	Command bson.D
}

func main() {
	ctx := context.Background()
	forceReset := flag.Bool("reset", false, "drop and reseed both engines before benchmarking")
	flag.Parse()

	_ = os.MkdirAll("plans/postgres", 0o755)
	_ = os.MkdirAll("plans/mongo", 0o755)

	fixture, err := loadFixture(resolveFixturePath("fixture/fixture.json"))
	if err != nil {
		panic(err)
	}
	if len(fixture.Products) == 0 {
		panic("fixture contains no products")
	}

	spotCheckProduct := fixture.Products[0]

	if _, err := bootstrap.MigrateAndSeedWithOptions(ctx, resolveFixturePath("fixture/fixture.json"), bootstrap.SeedOptions{ForceReset: *forceReset}); err != nil {
		panic(err)
	}

	pg, err := connectPostgres(ctx)
	if err != nil {
		panic(err)
	}
	defer pg.Close(ctx)

	mongoClient, err := connectMongo(ctx)
	if err != nil {
		panic(err)
	}
	defer mongoClient.Disconnect(ctx)

	db := mongoClient.Database("benchmark")

	postgresQueries := []postgresQuery{
		{Name: "point_lookup", File: "plans/postgres/point_lookup.json", SQL: `SELECT * FROM products WHERE id = $1`, Args: []any{spotCheckProduct.ID}},
		{Name: "range_scan", File: "plans/postgres/range_scan.json", SQL: `SELECT * FROM products WHERE price BETWEEN 100 AND 500`},
		{Name: "aggregate", File: "plans/postgres/aggregate.json", SQL: `SELECT category_id, COUNT(*), AVG(price) FROM products GROUP BY category_id`},
		{Name: "join_lookup", File: "plans/postgres/join.json", SQL: `SELECT p.id, p.name, c.name FROM products p JOIN categories c ON p.category_id = c.id`},
	}

	mongoQueries := []mongoQuery{
		{Name: "point_lookup", File: "plans/mongo/point_lookup.json", Command: bson.D{{"explain", bson.D{{"find", "products"}, {"filter", bson.D{{"id", spotCheckProduct.ID}}}}}, {"verbosity", "executionStats"}}},
		{Name: "range_scan", File: "plans/mongo/range_scan.json", Command: bson.D{{"explain", bson.D{{"find", "products"}, {"filter", bson.D{{"price", bson.D{{"$gte", 100}, {"$lte", 500}}}}}}}, {"verbosity", "executionStats"}}},
		{Name: "aggregate", File: "plans/mongo/aggregate.json", Command: bson.D{{"explain", bson.D{{"aggregate", "products"}, {"pipeline", bson.A{bson.D{{"$group", bson.D{{"_id", "$categoryId"}, {"count", bson.D{{"$sum", 1}}}, {"avgPrice", bson.D{{"$avg", "$price"}}}}}}}}, {"cursor", bson.D{}}}}, {"verbosity", "executionStats"}}},
		{Name: "join_lookup", File: "plans/mongo/lookup.json", Command: bson.D{{"explain", bson.D{{"aggregate", "products"}, {"pipeline", bson.A{bson.D{{"$lookup", bson.D{{"from", "categories"}, {"localField", "categoryId"}, {"foreignField", "_id"}, {"as", "category"}}}}}}, {"cursor", bson.D{}}}}, {"verbosity", "executionStats"}}},
	}

	results, err := runPostgresBenchmark(ctx, pg, postgresQueries)
	if err != nil {
		panic(err)
	}

	mongoResults, err := runMongoBenchmark(ctx, db, mongoQueries)
	if err != nil {
		panic(err)
	}
	results = append(results, mongoResults...)

	spotCheck := performFixtureSpotCheck(ctx, pg, db, spotCheckProduct)
	fmt.Printf("fixture spot check: %+v\n", spotCheck)

	report := make([]map[string]any, 0, len(results))
	for _, result := range results {
		report = append(report, map[string]any{
			"shape":  strings.ToLower(result.Shape),
			"engine": result.Engine,
			"timeMs": result.TimeMs,
		})
	}

	reportOut, err := json.MarshalIndent(map[string]any{"fixtureSpotCheck": spotCheck, "results": report}, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile("benchmark-results.json", reportOut, 0o644); err != nil {
		panic(err)
	}
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

func runPostgresBenchmark(ctx context.Context, pg *pgx.Conn, queries []postgresQuery) ([]Result, error) {
	results := make([]Result, 0, len(queries))
	for _, query := range queries {
		var plan string
		if err := pg.QueryRow(ctx, "EXPLAIN (ANALYZE, FORMAT JSON) "+query.SQL, query.Args...).Scan(&plan); err != nil {
			return nil, err
		}
		if err := os.WriteFile(query.File, []byte(plan), 0o644); err != nil {
			return nil, err
		}

		var parsed []map[string]any
		if err := json.Unmarshal([]byte(plan), &parsed); err != nil {
			return nil, err
		}

		results = append(results, Result{Shape: query.Name, Engine: "postgres", TimeMs: extractPostgresExecutionTime(parsed)})
	}
	return results, nil
}

func runMongoBenchmark(ctx context.Context, db *mongo.Database, queries []mongoQuery) ([]Result, error) {
	results := make([]Result, 0, len(queries))
	for _, query := range queries {
		var result bson.M
		if err := db.RunCommand(ctx, query.Command).Decode(&result); err != nil {
			return nil, err
		}

		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(query.File, data, 0o644); err != nil {
			return nil, err
		}

		execTime := extractMongoExecutionTime(result)
		fmt.Printf("mongo shape=%s execTime=%.2f\n", query.Name, execTime)
		results = append(results, Result{Shape: query.Name, Engine: "mongo", TimeMs: execTime})
	}
	return results, nil
}

func extractPostgresExecutionTime(parsed []map[string]any) float64 {
	if len(parsed) == 0 {
		return 0
	}
	if executionTime, ok := parsed[0]["Execution Time"].(float64); ok {
		return executionTime
	}
	return 0
}

func extractMongoExecutionTime(result bson.M) float64 {
	executionStats, ok := result["executionStats"]
	if !ok {
		return 0
	}

	switch stats := executionStats.(type) {
	case bson.D:
		for _, elem := range stats {
			if elem.Key != "executionTimeMillis" {
				continue
			}
			switch value := elem.Value.(type) {
			case int:
				return float64(value)
			case int32:
				return float64(value)
			case int64:
				return float64(value)
			case float64:
				return value
			}
		}
	case bson.M:
		return extractMongoExecutionTimeFromMap(stats)
	case map[string]any:
		return extractMongoExecutionTimeFromMap(stats)
	default:
		value := reflect.ValueOf(executionStats)
		if value.Kind() != reflect.Map || value.Type().Key().Kind() != reflect.String {
			return 0
		}
		convertedStats := make(map[string]any)
		iter := value.MapRange()
		for iter.Next() {
			convertedStats[iter.Key().String()] = iter.Value().Interface()
		}
		return extractMongoExecutionTimeFromMap(convertedStats)
	}

	return 0
}

func extractMongoExecutionTimeFromMap(stats map[string]any) float64 {
	switch value := stats["executionTimeMillis"].(type) {
	case int:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	case float64:
		return value
	default:
		return 0
	}
}

func performFixtureSpotCheck(ctx context.Context, pg *pgx.Conn, db *mongo.Database, product models.Product) FixtureCheck {
	var pgName string
	var pgPrice float64
	if err := pg.QueryRow(ctx, `SELECT name, price FROM products WHERE id = $1`, product.ID).Scan(&pgName, &pgPrice); err != nil {
		panic(err)
	}

	var mongoDoc bson.M
	if err := db.Collection("products").FindOne(ctx, bson.M{"id": product.ID}).Decode(&mongoDoc); err != nil {
		panic(err)
	}

	mongoName, _ := mongoDoc["name"].(string)
	mongoPrice, _ := mongoDoc["price"].(float64)

	return FixtureCheck{
		FixtureID:  product.ID,
		PGName:     pgName,
		MongoName:  mongoName,
		PGPrice:    pgPrice,
		MongoPrice: mongoPrice,
		Match:      pgName == mongoName && pgPrice == mongoPrice,
	}
}
