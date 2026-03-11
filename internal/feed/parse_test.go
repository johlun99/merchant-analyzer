package feed_test

import (
	"os"
	"testing"

	"github.com/johlun99/merchant-analyzer/internal/feed"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("../../testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestParseValidFeed(t *testing.T) {
	data := readFixture(t, "valid_feed.xml")

	f, err := feed.Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.ProductCount != 5 {
		t.Errorf("ProductCount = %d, want 5", f.ProductCount)
	}
	if len(f.Products) != 5 {
		t.Fatalf("len(Products) = %d, want 5", len(f.Products))
	}

	p := f.Products[0]
	if p.ID != "prod-001" {
		t.Errorf("ID = %q, want %q", p.ID, "prod-001")
	}
	if p.Title != "Blue Running Shoes" {
		t.Errorf("Title = %q, want %q", p.Title, "Blue Running Shoes")
	}
	if p.Price != "89.99 SEK" {
		t.Errorf("Price = %q, want %q", p.Price, "89.99 SEK")
	}
	if p.Brand != "RunFast" {
		t.Errorf("Brand = %q, want %q", p.Brand, "RunFast")
	}
	if p.Color != "Blue" {
		t.Errorf("Color = %q, want %q", p.Color, "Blue")
	}
	if p.Material != "Mesh" {
		t.Errorf("Material = %q, want %q", p.Material, "Mesh")
	}
	if len(p.AdditionalImages) == 0 {
		t.Error("AdditionalImages should not be empty")
	}
}

func TestParseStripsNamespacePrefixes(t *testing.T) {
	data := readFixture(t, "valid_feed.xml")

	f, err := feed.Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Fields should be populated despite g: namespace prefix in XML
	p := f.Products[0]
	if p.ID == "" {
		t.Error("ID should be populated (g:id namespace should be stripped)")
	}
	if p.GTIN == "" {
		t.Error("GTIN should be populated (g:gtin namespace should be stripped)")
	}
}

func TestParseMinimalFeed(t *testing.T) {
	data := readFixture(t, "minimal_feed.xml")

	f, err := feed.Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.ProductCount != 3 {
		t.Errorf("ProductCount = %d, want 3", f.ProductCount)
	}

	p := f.Products[0]
	if p.ID != "min-001" {
		t.Errorf("ID = %q, want min-001", p.ID)
	}
	// Optional fields should be empty
	if p.Brand != "" {
		t.Errorf("Brand should be empty for minimal feed, got %q", p.Brand)
	}
	if p.Color != "" {
		t.Errorf("Color should be empty for minimal feed, got %q", p.Color)
	}
}

func TestParseBrokenFeedIsIncomplete(t *testing.T) {
	// Broken feed has no closing root tag — Parse should return an error
	_, err := feed.Parse(readFixture(t, "broken_feed.xml"))
	if err == nil {
		t.Error("expected error for truncated feed, got nil")
	}
}

func TestParseEmptyFields(t *testing.T) {
	// Use a well-formed feed with empty required fields:
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:g="http://base.google.com/ns/1.0">
  <channel>
    <item>
      <g:id></g:id>
      <g:title>Empty ID Product</g:title>
      <g:price></g:price>
    </item>
  </channel>
</rss>`)

	f, err := feed.Parse(xmlData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.Products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(f.Products))
	}
	if f.Products[0].ID != "" {
		t.Errorf("ID should be empty, got %q", f.Products[0].ID)
	}
	if f.Products[0].Price != "" {
		t.Errorf("Price should be empty, got %q", f.Products[0].Price)
	}
}
