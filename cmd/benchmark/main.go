package main

import (
	"context"
	"encoding/json"
	"os"

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

func main() {
	ctx := context.Background()

	_ = os.MkdirAll("plans/postgres", 0755)
	_ = os.MkdirAll("plans/mongo", 0755)

	// postgres
	pg, err := pgx.Connect(
		ctx,
		"postgres://postgres:postgres@localhost:5432/benchmark",
	)
	if err != nil {
		panic(err)
	}
	defer pg.Close(ctx)

	// mongo
	mongoClient, err := mongo.Connect(
		options.Client().ApplyURI("mongodb://localhost:27017"),
	)
	if err != nil {
		panic(err)
	}
	defer mongoClient.Disconnect(ctx)

	db := mongoClient.Database("benchmark")

	// sample id
	var productID string
	err = pg.QueryRow(
		ctx,
		`select id from products limit 1`,
	).Scan(&productID)

	if err != nil {
		panic(err)
	}

	results := make([]Result, 0)

	// ======================
	// POSTGRES
	// ======================

	pgQueries := []struct {
		Name string
		File string
		SQL  string
		Args []any
	}{
		{
			"point_lookup",
			"plans/postgres/point_lookup.json",
			`SELECT * FROM products WHERE id = $1`,
			[]any{productID},
		},
		{
			"range_scan",
			"plans/postgres/range_scan.json",
			`SELECT * FROM products WHERE price BETWEEN 100 AND 500`,
			nil,
		},
		{
			"aggregate",
			"plans/postgres/aggregate.json",
			`SELECT category_id,COUNT(*),AVG(price)
			 FROM products
			 GROUP BY category_id`,
			nil,
		},
		{
			"join",
			"plans/postgres/join.json",
			`SELECT p.id,p.name,c.name
			 FROM products p
			 JOIN categories c
			 ON p.category_id = c.id`,
			nil,
		},
	}

	for _, q := range pgQueries {

		var plan string

		err := pg.QueryRow(
			ctx,
			"EXPLAIN (ANALYZE, FORMAT JSON) "+q.SQL,
			q.Args...,
		).Scan(&plan)

		if err != nil {
			panic(err)
		}

		_ = os.WriteFile(
			q.File,
			[]byte(plan),
			0644,
		)

		var parsed []map[string]any
		_ = json.Unmarshal([]byte(plan), &parsed)

		execTime := 0.0

		if len(parsed) > 0 {
			if t, ok := parsed[0]["Execution Time"].(float64); ok {
				execTime = t
			}
		}

		results = append(results, Result{
			Shape:  q.Name,
			Engine: "postgres",
			TimeMs: execTime,
		})
	}

	// ======================
	// MONGO
	// ======================

	mongoQueries := []struct {
		Name string
		File string
		Cmd  bson.D
	}{
		{
			Name: "point_lookup",
			File: "plans/mongo/point_lookup.json",
			Cmd: bson.D{
				{"explain", bson.D{
					{"find", "products"},
					{"filter", bson.D{
						{"id", productID},
					}},
				}},
				{"verbosity", "executionStats"},
			},
		},
		{
			Name: "range_scan",
			File: "plans/mongo/range_scan.json",
			Cmd: bson.D{
				{"explain", bson.D{
					{"find", "products"},
					{"filter", bson.D{
						{"price", bson.D{
							{"$gte", 100},
							{"$lte", 500},
						}},
					}},
				}},
				{"verbosity", "executionStats"},
			},
		},
		{
			Name: "aggregate",
			File: "plans/mongo/aggregate.json",
			Cmd: bson.D{
				{"explain", bson.D{
					{"aggregate", "products"},
					{"pipeline", bson.A{
						bson.D{
							{"$group", bson.D{
								{"_id", "$categoryId"},
								{"count", bson.D{
									{"$sum", 1},
								}},
								{"avgPrice", bson.D{
									{"$avg", "$price"},
								}},
							}},
						},
					}},
					{"cursor", bson.D{}},
				}},
				{"verbosity", "executionStats"},
			},
		},
		{
			Name: "lookup",
			File: "plans/mongo/lookup.json",
			Cmd: bson.D{
				{"explain", bson.D{
					{"aggregate", "products"},
					{"pipeline", bson.A{
						bson.D{
							{"$lookup", bson.D{
								{"from", "categories"},
								{"localField", "categoryId"},
								{"foreignField", "_id"},
								{"as", "category"},
							}},
						},
					}},
					{"cursor", bson.D{}},
				}},
				{"verbosity", "executionStats"},
			},
		},
	}

	for _, q := range mongoQueries {

		var result bson.M

		err := db.RunCommand(
			ctx,
			q.Cmd,
		).Decode(&result)

		if err != nil {
			panic(err)
		}

		data, _ := json.MarshalIndent(
			result,
			"",
			"  ",
		)

		_ = os.WriteFile(
			q.File,
			data,
			0644,
		)

		execTime := 0.0

		if stats, ok := result["executionStats"].(bson.M); ok {
			switch v := stats["executionTimeMillis"].(type) {
			case int32:
				execTime = float64(v)
			case int64:
				execTime = float64(v)
			case float64:
				execTime = v
			}
		}

		results = append(results, Result{
			Shape:  q.Name,
			Engine: "mongo",
			TimeMs: execTime,
		})
	}

	out, _ := json.MarshalIndent(
		results,
		"",
		"  ",
	)

	_ = os.WriteFile(
		"benchmark-results.json",
		out,
		0644,
	)
}
