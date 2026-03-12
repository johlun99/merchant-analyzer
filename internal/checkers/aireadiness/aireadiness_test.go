package aireadiness_test

import (
	"context"
	"fmt"
	"os"
	"strings"
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

func TestAIReadinessUCPExamplesPopulated(t *testing.T) {
	products := []feed.Product{
		{ID: "p1", Title: "Shirt", Description: "A shirt.", Price: "10", Availability: "in stock",
			Link: "https://x.com", ImageLink: "https://x.com/img.jpg", Condition: "new"},
		// brand and gtin_or_mpn missing → UCP items should have examples
	}
	f := &feed.Feed{Products: products, ProductCount: 1}
	result := aireadiness.NewChecker().Run(context.Background(), f)

	for _, item := range result.Items {
		if item.Field == "brand" {
			if len(item.Examples) == 0 {
				t.Error("expected examples for brand UCP violation, got none")
			}
			return
		}
	}
	t.Error("expected item for field \"brand\"")
}

func TestAIReadinessLLMExamplesShowDescriptionLength(t *testing.T) {
	products := []feed.Product{
		{ID: "p1", Title: "Shirt", Description: "Short.", Price: "10", Availability: "in stock",
			Link: "https://x.com", ImageLink: "https://x.com/img.jpg"},
	}
	f := &feed.Feed{Products: products, ProductCount: 1}
	result := aireadiness.NewChecker().Run(context.Background(), f)

	for _, item := range result.Items {
		if item.Field == "description_length" {
			if len(item.Examples) == 0 {
				t.Fatal("expected examples for description_length violation, got none")
			}
			ex := item.Examples[0]
			if !strings.Contains(ex, "chars") {
				t.Errorf("expected example to contain char count, got %q", ex)
			}
			return
		}
	}
	t.Error("expected item for field \"description_length\"")
}

func TestAIReadinessExamplesCappedAt10(t *testing.T) {
	products := make([]feed.Product, 15)
	for i := range products {
		products[i] = feed.Product{
			ID: fmt.Sprintf("p%d", i+1), Title: "Shirt", Description: "desc",
			Price: "10", Availability: "in stock", Link: "https://x.com", ImageLink: "https://x.com/img.jpg",
			// no color → LLM color violation for all 15
		}
	}
	f := &feed.Feed{Products: products, ProductCount: len(products)}
	result := aireadiness.NewChecker().Run(context.Background(), f)

	for _, item := range result.Items {
		if item.Field == "color" {
			if len(item.Examples) == 0 {
				t.Error("expected examples, got none")
			}
			if len(item.Examples) > 10 {
				t.Errorf("examples capped at 10, got %d", len(item.Examples))
			}
			return
		}
	}
	t.Error("expected item for field \"color\"")
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

func TestAIReadinessItemsHaveImpact(t *testing.T) {
	// A feed that triggers at least one AI Readiness finding.
	// Every item with a finding must have a non-empty Impact.
	products := []feed.Product{
		{
			ID:    "p001",
			Title: "Test Product",
			// Missing color, size, material, short description, no additional image
			Description:  "Short.",
			Price:        "9.99 SEK",
			Availability: "in stock",
			Link:         "https://example.com/p001",
			ImageLink:    "https://example.com/img.jpg",
			Brand:        "ACME",
			GTIN:         "12345678",
			Condition:    "new",
		},
	}
	f := &feed.Feed{Products: products, ProductCount: 1}
	c := aireadiness.NewChecker()
	result := c.Run(context.Background(), f)

	if len(result.Items) == 0 {
		t.Fatal("expected at least one finding")
	}
	for _, item := range result.Items {
		if item.Impact == "" {
			t.Errorf("item %q has empty Impact; want High/Medium/Low", item.Field)
		}
		if item.ImpactDesc == "" {
			t.Errorf("item %q has empty ImpactDesc", item.Field)
		}
	}
}

func TestAIReadinessImpactValuesAreValid(t *testing.T) {
	// Impact must be one of the three recognised levels.
	products := []feed.Product{
		{ID: "p001", Title: "T", Description: "D", Price: "9.99 SEK", Availability: "in stock",
			Link: "https://example.com", ImageLink: "https://example.com/img.jpg"},
	}
	f := &feed.Feed{Products: products, ProductCount: 1}
	result := aireadiness.NewChecker().Run(context.Background(), f)

	valid := map[string]bool{"High": true, "Medium": true, "Low": true}
	for _, item := range result.Items {
		if item.Impact != "" && !valid[item.Impact] {
			t.Errorf("item %q has invalid Impact %q; want High/Medium/Low", item.Field, item.Impact)
		}
	}
}
