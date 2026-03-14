package attributes_test

import (
	"context"
	"fmt"
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

func TestAttributeCheckerAffectedProductsPopulated(t *testing.T) {
	products := []feed.Product{
		{ID: "sku-001", Title: "Product A", Description: "", Price: "10", Availability: "in stock", Link: "http://x.com", ImageLink: "http://x.com/img.jpg"},
		{ID: "sku-002", Title: "Product B", Description: "Desc", Price: "10", Availability: "in stock", Link: "http://x.com", ImageLink: "http://x.com/img.jpg"},
	}
	f := &feed.Feed{Products: products, ProductCount: 2}
	c := attributes.NewChecker()
	result := c.Run(context.Background(), f)

	var descItem *checker.Item
	for i := range result.Items {
		if result.Items[i].Field == "description" {
			descItem = &result.Items[i]
			break
		}
	}
	if descItem == nil {
		t.Fatal("expected item for field 'description'")
	}
	if len(descItem.AffectedProducts) != 1 {
		t.Fatalf("AffectedProducts len = %d, want 1", len(descItem.AffectedProducts))
	}
	if descItem.AffectedProducts[0].ID != "sku-001" {
		t.Errorf("AffectedProducts[0].ID = %q, want %q", descItem.AffectedProducts[0].ID, "sku-001")
	}
	if descItem.AffectedProducts[0].Title != "Product A" {
		t.Errorf("AffectedProducts[0].Title = %q, want %q", descItem.AffectedProducts[0].Title, "Product A")
	}
}

func TestAttributeCheckerAffectedProductsMatchCount(t *testing.T) {
	// All 3 products missing 'brand' is not a required field, skip — use 'description'
	products := make([]feed.Product, 5)
	for i := range products {
		products[i] = feed.Product{
			ID: fmt.Sprintf("sku-%d", i), Title: fmt.Sprintf("Prod %d", i),
			Price: "10", Availability: "in stock", Link: "http://x.com", ImageLink: "http://x.com/img.jpg",
		}
	}
	// 3 of 5 missing description
	products[0].Description = "Has desc"
	products[1].Description = "Has desc"

	f := &feed.Feed{Products: products, ProductCount: 5}
	c := attributes.NewChecker()
	result := c.Run(context.Background(), f)

	for _, item := range result.Items {
		if item.Field == "description" {
			if item.Count != len(item.AffectedProducts) {
				t.Errorf("Count = %d, len(AffectedProducts) = %d; they must match", item.Count, len(item.AffectedProducts))
			}
			return
		}
	}
	t.Error("no description item found")
}
