package main

import (
	"encoding/json"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"

	"benchmark/models"
)

func main() {
	r := rand.New(rand.NewSource(42))

	categories := []models.Category{
		{ID: 1, Name: "Laptop"},
		{ID: 2, Name: "Phone"},
		{ID: 3, Name: "Book"},
		{ID: 4, Name: "Accessory"},
		{ID: 5, Name: "Gaming"},
	}

	tags := []string{
		"new",
		"sale",
		"popular",
		"premium",
		"office",
		"student",
		"gaming",
	}

	products := make([]models.Product, 0, 10000)

	for i := 0; i < 10000; i++ {

		category := categories[r.Intn(len(categories))]

		products = append(products, models.Product{
			ID:         uuid.NewString(),
			Name:       "Product-" + uuid.NewString()[:8],
			CategoryID: category.ID,
			Price:      float64(r.Intn(100000)) / 100,
			Tags: []string{
				tags[r.Intn(len(tags))],
				tags[r.Intn(len(tags))],
			},
			CreatedAt: time.Now().
				Add(-time.Duration(r.Intn(365*24)) * time.Hour),
		})
	}

	fixture := models.Fixture{
		Categories: categories,
		Products:   products,
	}

	data, err := json.MarshalIndent(
		fixture,
		"",
		"  ",
	)
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll("fixture", 0755)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(
		"fixture/fixture.json",
		data,
		0644,
	)
	if err != nil {
		panic(err)
	}
}
