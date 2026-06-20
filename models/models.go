package models

import "time"

type Fixture struct {
	Categories []Category `json:"categories"`
	Products   []Product  `json:"products"`
}

type Category struct {
	ID   int    `json:"id" bson:"_id"`
	Name string `json:"name" bson:"name"`
}

type Product struct {
	ID         string    `json:"id" bson:"id"`
	Name       string    `json:"name" bson:"name"`
	CategoryID int       `json:"categoryId" bson:"categoryId"`
	Price      float64   `json:"price" bson:"price"`
	Tags       []string  `json:"tags" bson:"tags"`
	CreatedAt  time.Time `json:"createdAt" bson:"createdAt"`
}