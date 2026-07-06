package main

import (
	"benchmark/cmd/models"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

func main() {
	fixture := generateFixture(42, 10000)
	outPath := filepath.Join("cmd", "benchmark", "fixture", "fixture.json")
	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote fixture to %s with %d products\n", outPath, len(fixture.Products))
}

func generateFixture(seed int64, productCount int) models.Fixture {
	rng := rand.New(rand.NewSource(seed))
	categories := make([]models.Category, 0, 20)
	for i := 1; i <= 20; i++ {
		categories = append(categories, models.Category{ID: i, Name: fmt.Sprintf("Category-%d", i)})
	}

	products := make([]models.Product, 0, productCount)
	for i := 0; i < productCount; i++ {
		categoryID := 1 + (i % len(categories))
		price := float64(50 + (rng.Intn(950) + rng.Intn(50)))
		createdAt := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i%365)
		products = append(products, models.Product{
			ID:         uuid.NewString(),
			Name:       fmt.Sprintf("Product-%04d", i),
			CategoryID: categoryID,
			Price:      price,
			Tags:       []string{"tag-a", "tag-b"},
			CreatedAt:  createdAt,
		})
	}

	return models.Fixture{Categories: categories, Products: products}
}
