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
