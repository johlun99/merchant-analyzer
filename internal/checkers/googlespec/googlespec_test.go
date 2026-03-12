// Package googlespec_test tests the Google Feed Spec checker.
package googlespec_test

import (
	"context"
	"strings"
	"testing"

	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/checkers/googlespec"
	"github.com/johlun99/merchant-analyzer/internal/feed"
)

// feedFromItem wraps itemXML in a minimal RSS envelope and parses it.
func feedFromItem(t *testing.T, itemXML string) *feed.Feed {
	t.Helper()
	raw := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:g="http://base.google.com/ns/1.0">
  <channel>
    <item>` + itemXML + `</item>
  </channel>
</rss>`)
	f, err := feed.Parse(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	return f
}

func runChecker(t *testing.T, f *feed.Feed) checker.Result {
	t.Helper()
	return googlespec.NewChecker().Run(context.Background(), f)
}

// cleanItem contains all required + recommended fields with valid values.
// All violation tests include brand and google_product_category so the
// overall status is driven only by the field under test.
const cleanItem = `
      <g:id>prod-001</g:id>
      <g:title>Blue Running Shoes</g:title>
      <g:description>A great pair of running shoes for all terrain use.</g:description>
      <g:price>89.99 SEK</g:price>
      <g:availability>in stock</g:availability>
      <g:link>https://example.com/products/prod-001</g:link>
      <g:image_link>https://example.com/images/prod-001.jpg</g:image_link>
      <g:brand>RunFast</g:brand>
      <g:google_product_category>Shoes</g:google_product_category>
      <g:condition>new</g:condition>
      <g:gtin>1234567890123</g:gtin>
      <g:age_group>adult</g:age_group>
      <g:gender>unisex</g:gender>`

// requiredBase includes only the required fields + recommended presence fields.
// Use this as the base for violation tests so the only violations come from
// the field being tested.
const requiredBase = `
      <g:id>prod-001</g:id>
      <g:title>Blue Running Shoes</g:title>
      <g:description>A great pair of running shoes for all terrain use.</g:description>
      <g:price>89.99 SEK</g:price>
      <g:availability>in stock</g:availability>
      <g:link>https://example.com/products/prod-001</g:link>
      <g:image_link>https://example.com/images/prod-001.jpg</g:image_link>
      <g:brand>RunFast</g:brand>
      <g:google_product_category>Shoes</g:google_product_category>`

func TestGoogleSpecCleanFeedIsOK(t *testing.T) {
	f := feedFromItem(t, cleanItem)
	r := runChecker(t, f)
	if r.Status != checker.StatusOK {
		t.Errorf("status = %v, want StatusOK; items: %v", r.Status, r.Items)
	}
	if len(r.Items) != 0 {
		t.Errorf("expected 0 items, got %d: %v", len(r.Items), r.Items)
	}
}

func TestGoogleSpecEmptyFeedIsOK(t *testing.T) {
	r := runChecker(t, &feed.Feed{})
	if r.Status != checker.StatusOK {
		t.Errorf("status = %v, want StatusOK", r.Status)
	}
}

func TestGoogleSpecMissingRequiredField(t *testing.T) {
	// requiredBase minus price
	f := feedFromItem(t, `
      <g:id>prod-001</g:id>
      <g:title>Blue Running Shoes</g:title>
      <g:description>A great pair of shoes.</g:description>
      <g:availability>in stock</g:availability>
      <g:link>https://example.com/products/prod-001</g:link>
      <g:image_link>https://example.com/images/prod-001.jpg</g:image_link>
      <g:brand>RunFast</g:brand>
      <g:google_product_category>Shoes</g:google_product_category>`)
	r := runChecker(t, f)
	if r.Status != checker.StatusError {
		t.Errorf("status = %v, want StatusError", r.Status)
	}
	found := false
	for _, item := range r.Items {
		if item.Field == "price" {
			found = true
		}
	}
	if !found {
		t.Error("expected item for field \"price\"")
	}
}

func TestGoogleSpecInvalidPriceFormat(t *testing.T) {
	f := feedFromItem(t, `
      <g:id>prod-001</g:id>
      <g:title>Blue Running Shoes</g:title>
      <g:description>A great pair of shoes.</g:description>
      <g:price>9.99</g:price>
      <g:availability>in stock</g:availability>
      <g:link>https://example.com/products/prod-001</g:link>
      <g:image_link>https://example.com/images/prod-001.jpg</g:image_link>
      <g:brand>RunFast</g:brand>
      <g:google_product_category>Shoes</g:google_product_category>`)
	r := runChecker(t, f)
	if r.Status != checker.StatusError {
		t.Errorf("status = %v, want StatusError", r.Status)
	}
	found := false
	for _, item := range r.Items {
		if item.Field == "price" {
			found = true
		}
	}
	if !found {
		t.Error("expected item for field \"price\"")
	}
}

func TestGoogleSpecInvalidAvailability(t *testing.T) {
	f := feedFromItem(t, `
      <g:id>prod-001</g:id>
      <g:title>Blue Running Shoes</g:title>
      <g:description>A great pair of shoes.</g:description>
      <g:price>89.99 SEK</g:price>
      <g:availability>yes</g:availability>
      <g:link>https://example.com/products/prod-001</g:link>
      <g:image_link>https://example.com/images/prod-001.jpg</g:image_link>
      <g:brand>RunFast</g:brand>
      <g:google_product_category>Shoes</g:google_product_category>`)
	r := runChecker(t, f)
	if r.Status != checker.StatusError {
		t.Errorf("status = %v, want StatusError", r.Status)
	}
	found := false
	for _, item := range r.Items {
		if item.Field == "availability" {
			found = true
		}
	}
	if !found {
		t.Error("expected item for field \"availability\"")
	}
}

func TestGoogleSpecInvalidURL(t *testing.T) {
	f := feedFromItem(t, `
      <g:id>prod-001</g:id>
      <g:title>Blue Running Shoes</g:title>
      <g:description>A great pair of shoes.</g:description>
      <g:price>89.99 SEK</g:price>
      <g:availability>in stock</g:availability>
      <g:link>ftp://example.com/products/prod-001</g:link>
      <g:image_link>https://example.com/images/prod-001.jpg</g:image_link>
      <g:brand>RunFast</g:brand>
      <g:google_product_category>Shoes</g:google_product_category>`)
	r := runChecker(t, f)
	if r.Status != checker.StatusError {
		t.Errorf("status = %v, want StatusError", r.Status)
	}
	found := false
	for _, item := range r.Items {
		if item.Field == "link" {
			found = true
		}
	}
	if !found {
		t.Error("expected item for field \"link\"")
	}
}

func TestGoogleSpecIDTooLong(t *testing.T) {
	longID := strings.Repeat("a", 51)
	f := feedFromItem(t, `
      <g:id>`+longID+`</g:id>
      <g:title>Blue Running Shoes</g:title>
      <g:description>A great pair of shoes.</g:description>
      <g:price>89.99 SEK</g:price>
      <g:availability>in stock</g:availability>
      <g:link>https://example.com/products/prod-001</g:link>
      <g:image_link>https://example.com/images/prod-001.jpg</g:image_link>
      <g:brand>RunFast</g:brand>
      <g:google_product_category>Shoes</g:google_product_category>`)
	r := runChecker(t, f)
	if r.Status != checker.StatusError {
		t.Errorf("status = %v, want StatusError", r.Status)
	}
	found := false
	for _, item := range r.Items {
		if item.Field == "id" {
			found = true
		}
	}
	if !found {
		t.Error("expected item for field \"id\"")
	}
}

func TestGoogleSpecMissingBrand(t *testing.T) {
	// requiredBase minus brand
	f := feedFromItem(t, `
      <g:id>prod-001</g:id>
      <g:title>Blue Running Shoes</g:title>
      <g:description>A great pair of shoes.</g:description>
      <g:price>89.99 SEK</g:price>
      <g:availability>in stock</g:availability>
      <g:link>https://example.com/products/prod-001</g:link>
      <g:image_link>https://example.com/images/prod-001.jpg</g:image_link>
      <g:google_product_category>Shoes</g:google_product_category>`)
	r := runChecker(t, f)
	if r.Status != checker.StatusWarning {
		t.Errorf("status = %v, want StatusWarning", r.Status)
	}
	found := false
	for _, item := range r.Items {
		if item.Field == "brand" {
			found = true
		}
	}
	if !found {
		t.Error("expected item for field \"brand\"")
	}
}

func TestGoogleSpecMissingCategory(t *testing.T) {
	// requiredBase minus google_product_category
	f := feedFromItem(t, `
      <g:id>prod-001</g:id>
      <g:title>Blue Running Shoes</g:title>
      <g:description>A great pair of shoes.</g:description>
      <g:price>89.99 SEK</g:price>
      <g:availability>in stock</g:availability>
      <g:link>https://example.com/products/prod-001</g:link>
      <g:image_link>https://example.com/images/prod-001.jpg</g:image_link>
      <g:brand>RunFast</g:brand>`)
	r := runChecker(t, f)
	if r.Status != checker.StatusWarning {
		t.Errorf("status = %v, want StatusWarning", r.Status)
	}
	found := false
	for _, item := range r.Items {
		if item.Field == "google_product_category" {
			found = true
		}
	}
	if !found {
		t.Error("expected item for field \"google_product_category\"")
	}
}

func TestGoogleSpecInvalidGTIN(t *testing.T) {
	f := feedFromItem(t, requiredBase+`
      <g:gtin>123</g:gtin>`)
	r := runChecker(t, f)
	if r.Status != checker.StatusWarning {
		t.Errorf("status = %v, want StatusWarning", r.Status)
	}
	found := false
	for _, item := range r.Items {
		if item.Field == "gtin" {
			found = true
		}
	}
	if !found {
		t.Error("expected item for field \"gtin\"")
	}
}

func TestGoogleSpecInvalidCondition(t *testing.T) {
	f := feedFromItem(t, requiredBase+`
      <g:condition>like-new</g:condition>`)
	r := runChecker(t, f)
	if r.Status != checker.StatusWarning {
		t.Errorf("status = %v, want StatusWarning", r.Status)
	}
	found := false
	for _, item := range r.Items {
		if item.Field == "condition" {
			found = true
		}
	}
	if !found {
		t.Error("expected item for field \"condition\"")
	}
}

func TestGoogleSpecInvalidAgeGroup(t *testing.T) {
	f := feedFromItem(t, requiredBase+`
      <g:age_group>teen</g:age_group>`)
	r := runChecker(t, f)
	if r.Status != checker.StatusWarning {
		t.Errorf("status = %v, want StatusWarning", r.Status)
	}
	found := false
	for _, item := range r.Items {
		if item.Field == "age_group" {
			found = true
		}
	}
	if !found {
		t.Error("expected item for field \"age_group\"")
	}
}

func TestGoogleSpecInvalidGender(t *testing.T) {
	f := feedFromItem(t, requiredBase+`
      <g:gender>other</g:gender>`)
	r := runChecker(t, f)
	if r.Status != checker.StatusWarning {
		t.Errorf("status = %v, want StatusWarning", r.Status)
	}
	found := false
	for _, item := range r.Items {
		if item.Field == "gender" {
			found = true
		}
	}
	if !found {
		t.Error("expected item for field \"gender\"")
	}
}
