package attributes_test

import (
	"context"
	"os"
	"testing"

	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/checkers/attributes"
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

func TestAttributeCheckerName(t *testing.T) {
	c := attributes.NewChecker()
	if c.Name() == "" {
		t.Error("Name() should not be empty")
	}
}

func TestAttributeCheckerValidFeedIsOK(t *testing.T) {
	c := attributes.NewChecker()
	f := parseFeed(t, "valid_feed.xml")

	result := c.Run(context.Background(), f)

	if result.Status != checker.StatusOK {
		t.Errorf("Status = %v, want StatusOK; items: %v", result.Status, result.Items)
	}
}

func TestAttributeCheckerMinimalFeedReportsError(t *testing.T) {
	c := attributes.NewChecker()
	f := parseFeed(t, "minimal_feed.xml")

	result := c.Run(context.Background(), f)

	// Minimal feed missing description, availability, link, image_link on all 3 products = 100% missing
	if result.Status != checker.StatusError {
		t.Errorf("Status = %v, want StatusError for feed missing many required fields", result.Status)
	}
	if len(result.Items) == 0 {
		t.Error("Items should list fields with high missing rate")
	}
}

func TestAttributeCheckerReportsWarningForLowMissRate(t *testing.T) {
	// 1 of 20 products missing description = 5% → Warning threshold
	products := make([]feed.Product, 20)
	for i := range products {
		products[i] = feed.Product{
			ID:           "p",
			Title:        "T",
			Description:  "D",
			Price:        "10",
			Availability: "in stock",
			Link:         "https://x.com",
			ImageLink:    "https://x.com/img.jpg",
		}
	}
	products[0].Description = "" // 1/20 = 5% missing

	f := &feed.Feed{Products: products, ProductCount: 20}
	c := attributes.NewChecker()
	result := c.Run(context.Background(), f)

	hasDescriptionItem := false
	for _, item := range result.Items {
		if item.Field == "description" {
			hasDescriptionItem = true
		}
	}
	if !hasDescriptionItem {
		t.Error("Items should include description field finding")
	}
}

func TestAttributeCheckerReportsOKWhenNoneMissing(t *testing.T) {
	f := parseFeed(t, "ai_optimized_feed.xml")
	c := attributes.NewChecker()
	result := c.Run(context.Background(), f)

	if result.Status != checker.StatusOK {
		t.Errorf("Status = %v, want StatusOK; items: %v", result.Status, result.Items)
	}
}

func TestAttributeCheckerEmptyFeed(_ *testing.T) {
	f := &feed.Feed{Products: nil, ProductCount: 0}
	c := attributes.NewChecker()
	result := c.Run(context.Background(), f)

	// No products → can't compute coverage → should not panic, status OK or Warning
	_ = result
}
