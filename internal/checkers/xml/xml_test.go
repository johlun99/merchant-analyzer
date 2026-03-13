package xmlcheck_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/johlun99/merchant-analyzer/internal/checker"
	xmlcheck "github.com/johlun99/merchant-analyzer/internal/checkers/xml"
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
	f.URL = "test://" + fixture
	return f
}

func feedFromBytes(data []byte) *feed.Feed {
	f, _ := feed.Parse(data)
	if f == nil {
		f = &feed.Feed{}
	}
	f.Raw = data
	return f
}

func TestXMLCheckerNameIsSet(t *testing.T) {
	c := xmlcheck.NewChecker()
	if c.Name() == "" {
		t.Error("Name() should not be empty")
	}
}

func TestXMLCheckerValidFeedIsOK(t *testing.T) {
	c := xmlcheck.NewChecker()
	f := parseFeed(t, "valid_feed.xml")

	result := c.Run(context.Background(), f)

	if result.Status != checker.StatusOK {
		t.Errorf("Status = %v, want StatusOK; items: %v", result.Status, result.Items)
	}
}

func TestXMLCheckerMinimalFeedReportsWarning(t *testing.T) {
	c := xmlcheck.NewChecker()
	f := parseFeed(t, "minimal_feed.xml")

	result := c.Run(context.Background(), f)

	// Minimal feed has products with empty description, availability, link, image_link
	if result.Status != checker.StatusWarning {
		t.Errorf("Status = %v, want StatusWarning for feed with empty required fields", result.Status)
	}
	if len(result.Items) == 0 {
		t.Error("Items should list the empty required fields")
	}
}

func TestXMLCheckerBrokenFeedReportsError(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/broken_feed.xml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	c := xmlcheck.NewChecker()
	f := feedFromBytes(data)
	result := c.Run(context.Background(), f)

	if result.Status != checker.StatusError {
		t.Errorf("Status = %v, want StatusError for truncated feed", result.Status)
	}
	if len(result.Items) == 0 {
		t.Error("Items should contain at least one finding")
	}
}

func TestXMLCheckerDetectsEmptyRequiredAttributes(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:g="http://base.google.com/ns/1.0">
  <channel>
    <item>
      <g:id></g:id>
      <g:title>Product</g:title>
      <g:price></g:price>
    </item>
  </channel>
</rss>`)

	c := xmlcheck.NewChecker()
	result := c.Run(context.Background(), feedFromBytes(data))

	if result.Status != checker.StatusWarning && result.Status != checker.StatusError {
		t.Errorf("Status = %v, want Warning or Error for empty required fields", result.Status)
	}
	if len(result.Items) == 0 {
		t.Error("Items should list the empty fields found")
	}
}

func TestXMLCheckerDetectsExcessivelyLongValue(t *testing.T) {
	longValue := strings.Repeat("x", 5001)
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:g="http://base.google.com/ns/1.0">
  <channel>
    <item>
      <g:id>p1</g:id>
      <g:title>` + longValue + `</g:title>
    </item>
  </channel>
</rss>`)

	c := xmlcheck.NewChecker()
	result := c.Run(context.Background(), feedFromBytes(data))

	if result.Status == checker.StatusOK {
		t.Error("Status should not be OK for value >5000 chars")
	}
}

// TestXMLCheckerAggregatesPerField verifies that items are one-per-field, not one-per-product.
func TestXMLCheckerAggregatesPerField(t *testing.T) {
	// Two products both missing 'id' and 'price' — should produce 2 items (one per field),
	// not 4 items (one per product-field pair).
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:g="http://base.google.com/ns/1.0">
  <channel>
    <item>
      <g:id></g:id>
      <g:title>Product A</g:title>
      <g:description>Desc A</g:description>
      <g:price></g:price>
      <g:availability>in stock</g:availability>
      <g:link>http://example.com/a</g:link>
      <g:image_link>http://example.com/a.jpg</g:image_link>
    </item>
    <item>
      <g:id></g:id>
      <g:title>Product B</g:title>
      <g:description>Desc B</g:description>
      <g:price></g:price>
      <g:availability>in stock</g:availability>
      <g:link>http://example.com/b</g:link>
      <g:image_link>http://example.com/b.jpg</g:image_link>
    </item>
  </channel>
</rss>`)

	c := xmlcheck.NewChecker()
	result := c.Run(context.Background(), feedFromBytes(data))

	// Expect exactly 2 items: one for "id", one for "price"
	if len(result.Items) != 2 {
		t.Errorf("Items count = %d, want 2 (one per field); items: %v", len(result.Items), result.Items)
	}
	for _, item := range result.Items {
		if item.Count != 2 {
			t.Errorf("Item %q Count = %d, want 2", item.Field, item.Count)
		}
	}
}

// TestXMLCheckerAffectedProductsPopulated verifies AffectedProducts is set correctly.
func TestXMLCheckerAffectedProductsPopulated(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:g="http://base.google.com/ns/1.0">
  <channel>
    <item>
      <g:id>sku-001</g:id>
      <g:title>Laptop</g:title>
      <g:description>A laptop</g:description>
      <g:price></g:price>
      <g:availability>in stock</g:availability>
      <g:link>http://example.com</g:link>
      <g:image_link>http://example.com/img.jpg</g:image_link>
    </item>
  </channel>
</rss>`)

	c := xmlcheck.NewChecker()
	result := c.Run(context.Background(), feedFromBytes(data))

	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	item := result.Items[0]
	if item.Field != "price" {
		t.Errorf("Field = %q, want %q", item.Field, "price")
	}
	if len(item.AffectedProducts) != 1 {
		t.Fatalf("AffectedProducts len = %d, want 1", len(item.AffectedProducts))
	}
	if item.AffectedProducts[0].ID != "sku-001" {
		t.Errorf("AffectedProducts[0].ID = %q, want %q", item.AffectedProducts[0].ID, "sku-001")
	}
	if item.AffectedProducts[0].Title != "Laptop" {
		t.Errorf("AffectedProducts[0].Title = %q, want %q", item.AffectedProducts[0].Title, "Laptop")
	}
}

// TestXMLCheckerDeterministicOrder verifies that items are always in the same field order.
func TestXMLCheckerDeterministicOrder(t *testing.T) {
	// All required fields missing — items should always come out in the same order.
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:g="http://base.google.com/ns/1.0">
  <channel>
    <item>
      <g:title>Only Title</g:title>
    </item>
  </channel>
</rss>`)

	c := xmlcheck.NewChecker()
	want := []string{"id", "description", "price", "availability", "link", "image_link"}

	for range 5 {
		result := c.Run(context.Background(), feedFromBytes(data))
		var fields []string
		for _, it := range result.Items {
			fields = append(fields, it.Field)
		}
		if len(fields) != len(want) {
			t.Fatalf("got %d items, want %d", len(fields), len(want))
		}
		for i, f := range fields {
			if f != want[i] {
				t.Errorf("run: item[%d].Field = %q, want %q", i, f, want[i])
			}
		}
	}
}

// TestXMLCheckerStructuralErrorHasNilAffectedProducts verifies that structural items have no products.
func TestXMLCheckerStructuralErrorHasNilAffectedProducts(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>`) // truncated

	c := xmlcheck.NewChecker()
	result := c.Run(context.Background(), feedFromBytes(data))

	for _, item := range result.Items {
		if item.Field == "xml" && item.AffectedProducts != nil {
			t.Error("structural xml item should have nil AffectedProducts")
		}
	}
}
