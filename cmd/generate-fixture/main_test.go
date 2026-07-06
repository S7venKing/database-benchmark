package main

import "testing"

func TestGenerateFixtureCreatesDeterministicDataset(t *testing.T) {
	fixture := generateFixture(42, 10000)

	if len(fixture.Categories) != 20 {
		t.Fatalf("expected 20 categories, got %d", len(fixture.Categories))
	}

	if len(fixture.Products) != 10000 {
		t.Fatalf("expected 10000 products, got %d", len(fixture.Products))
	}

	if fixture.Products[0].ID == "" {
		t.Fatalf("expected first product to have an id")
	}

	if fixture.Products[9999].CategoryID == 0 {
		t.Fatalf("expected last product to have a category id")
	}
}
