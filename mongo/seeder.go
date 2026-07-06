package mongo

import (
	"benchmark/cmd/models"
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Seeder struct {
	DB *mongo.Database
}

func (*Seeder) CreateSchema(
	ctx context.Context,
	db *mongo.Database,
) error {

	err := db.CreateCollection(ctx, "categories")
	if err != nil &&
		!mongo.IsDuplicateKeyError(err) {
		// ignore collection existed
	}

	err = db.CreateCollection(ctx, "products")
	if err != nil &&
		!mongo.IsDuplicateKeyError(err) {
		// ignore collection existed
	}

	_, err = db.Collection("products").
		Indexes().
		CreateMany(ctx, []mongo.IndexModel{
			{
				Keys: bson.D{
					{Key: "categoryId", Value: 1},
				},
			},
			{
				Keys: bson.D{
					{Key: "price", Value: 1},
				},
			},
		})

	return err
}

func (s *Seeder) Count(ctx context.Context) (int64, int64, error) {
	categoriesCount, err := s.DB.Collection("categories").CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, 0, err
	}

	productsCount, err := s.DB.Collection("products").CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, 0, err
	}

	return categoriesCount, productsCount, nil
}

func (s *Seeder) Seed(ctx context.Context, fixture models.Fixture) error {

	if err := s.DB.Collection("products").Drop(ctx); err != nil {
		return err
	}

	if err := s.DB.Collection("categories").Drop(ctx); err != nil {
		return err
	}

	categories := make([]interface{}, 0)

	for _, c := range fixture.Categories {
		categories = append(categories, c)
	}

	_, err := s.DB.
		Collection("categories").
		InsertMany(ctx, categories)

	if err != nil {
		return err
	}

	products := make([]interface{}, 0)

	for _, p := range fixture.Products {
		products = append(products, p)
	}

	_, err = s.DB.
		Collection("products").
		InsertMany(ctx, products)

	if err != nil {
		return err
	}

	_, err = s.DB.Collection("products").
		Indexes().
		CreateMany(ctx, []mongo.IndexModel{
			{
				Keys: map[string]int{
					"categoryId": 1,
				},
			},
			{
				Keys: map[string]int{
					"price": 1,
				},
			},
		})

	return err
}
