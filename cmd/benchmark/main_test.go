package main

import "testing"

func TestExtractPostgresExecutionTime(t *testing.T) {
	plans := []map[string]any{{"Execution Time": 2.377}}
	if got := extractPostgresExecutionTime(plans); got != 2.377 {
		t.Fatalf("expected 2.377, got %v", got)
	}
}

func TestExtractMongoExecutionTime(t *testing.T) {
	result := map[string]any{
		"executionStats": map[string]any{
			"executionTimeMillis": 4,
		},
	}
	if got := extractMongoExecutionTime(result); got != 4 {
		t.Fatalf("expected 4, got %v", got)
	}
}
