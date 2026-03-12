package aireadiness_test

import (
	"context"
	"os"
	"testing"

	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/checkers/aireadiness"
	"github.com/johlun99/merchant-analyzer/internal/feed"
)

func parseFeed(t *testing.T, fixture string) *feed.Feed {
	t.Helper()
	data, err := os.ReadFile("../../../testdata/" + fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	f, err := feed.Parse(data)
	if err != nil {
		t.Fatalf("parse fixture %s: %v", fixture, err)
	}
	f.Raw = data
	return f
}

func TestAIReadinessCheckerName(t *testing.T) {
	c := aireadiness.NewChecker()
	if c.Name() == "" {
		t.Error("Name() should not be empty")
	}
}

func TestAIReadinessCheckerScoreIsSet(t *testing.T) {
	c := aireadiness.NewChecker()
	f := parseFeed(t, "valid_feed.xml")

	result := c.Run(context.Background(), f)

	if result.Score == nil {
		t.Fatal("Score should not be nil")
	}
	if *result.Score < 0 || *result.Score > 100 {
		t.Errorf("Score = %d, want 0–100", *result.Score)
	}
}

func TestAIReadinessCheckerOptimizedFeedScores100(t *testing.T) {
	c := aireadiness.NewChecker()
	f := parseFeed(t, "ai_optimized_feed.xml")

	result := c.Run(context.Background(), f)

	if result.Score == nil {
		t.Fatal("Score should not be nil")
	}
	if *result.Score != 100 {
		t.Errorf("Score = %d, want 100 for fully optimized feed", *result.Score)
	}
}

func TestAIReadinessCheckerMinimalFeedScoresLow(t *testing.T) {
	c := aireadiness.NewChecker()
	f := parseFeed(t, "minimal_feed.xml")

	result := c.Run(context.Background(), f)

	if result.Score == nil {
		t.Fatal("Score should not be nil")
	}
	if *result.Score > 30 {
		t.Errorf("Score = %d, want ≤30 for minimal feed", *result.Score)
	}
}

func TestAIReadinessCheckerReportsItems(t *testing.T) {
	c := aireadiness.NewChecker()
	f := parseFeed(t, "minimal_feed.xml")

	result := c.Run(context.Background(), f)

	if len(result.Items) == 0 {
		t.Error("Items should list missing fields")
	}
}

func TestAIReadinessCheckerStatusReflectsScore(t *testing.T) {
	c := aireadiness.NewChecker()

	tests := []struct {
		fixture        string
		expectedStatus checker.Status
	}{
		{"ai_optimized_feed.xml", checker.StatusOK},
		{"minimal_feed.xml", checker.StatusError},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			f := parseFeed(t, tt.fixture)
			result := c.Run(context.Background(), f)
			if result.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v (score=%v)", result.Status, tt.expectedStatus, result.Score)
			}
		})
	}
}

func TestAIReadinessCheckerScoringFormula(t *testing.T) {
	// Feed with all UCP fields, no LLM attrs, no additional images
	// UCP score = 40, LLM = 0, Image = 15 (image_link present, no additional)
	products := []feed.Product{
		{
			ID:           "p1",
			Title:        "T",
			Description:  "Long enough description for the test to pass the 150 char check? No, only 60 chars.",
			Price:        "10",
			Availability: "in stock",
			Link:         "https://x.com",
			ImageLink:    "https://x.com/img.jpg",
			Brand:        "B",
			GTIN:         "123",
			Condition:    "new",
		},
	}
	f := &feed.Feed{Products: products, ProductCount: 1}
	c := aireadiness.NewChecker()
	result := c.Run(context.Background(), f)

	if result.Score == nil {
		t.Fatal("Score should not be nil")
	}
	// UCP: 10/10 fields present = 40pts
	// LLM: 0/4 (no color, size, material, desc too short) = 0pts
	// Image: image_link present = 15pts, no additional = 0pts
	// Total = 55
	expected := 55
	if *result.Score != expected {
		t.Errorf("Score = %d, want %d", *result.Score, expected)
	}
}
