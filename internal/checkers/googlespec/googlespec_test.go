// Package googlespec_test tests the Google Feed Spec checker.
package googlespec_test

import (
	"context"
	"fmt"
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
      <g:gender>unisex</g:gender>
      <g:mpn>RUN-001-BLU</g:mpn>
      <g:additional_image_link>https://example.com/images/prod-001-side.jpg</g:additional_image_link>
      <g:product_type>Footwear &gt; Running Shoes</g:product_type>`

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
      <g:google_product_category>Shoes</g:google_product_category>
      <g:mpn>RUN-001-BLU</g:mpn>
      <g:additional_image_link>https://example.com/images/prod-001-side.jpg</g:additional_image_link>
      <g:product_type>Footwear &gt; Running Shoes</g:product_type>`

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

// feedFromItems wraps multiple item XML strings in a minimal RSS envelope and parses it.
func feedFromItems(t *testing.T, itemsXML []string) *feed.Feed {
	t.Helper()
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:g="http://base.google.com/ns/1.0"><channel>`)
	for _, item := range itemsXML {
		sb.WriteString("<item>")
		sb.WriteString(item)
		sb.WriteString("</item>")
	}
	sb.WriteString("</channel></rss>")
	f, err := feed.Parse([]byte(sb.String()))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	return f
}

func TestGoogleSpecSubScoresPresent(t *testing.T) {
	f := feedFromItem(t, cleanItem)
	r := runChecker(t, f)
	if len(r.SubScores) != 3 {
		t.Fatalf("expected 3 sub-scores, got %d: %v", len(r.SubScores), r.SubScores)
	}
	want := []string{"Required", "Recommended", "Format"}
	for i, w := range want {
		if r.SubScores[i].Label != w {
			t.Errorf("SubScores[%d].Label = %q, want %q", i, r.SubScores[i].Label, w)
		}
	}
}

func TestGoogleSpecAllSubScores100OnCleanFeed(t *testing.T) {
	f := feedFromItem(t, cleanItem)
	r := runChecker(t, f)
	if len(r.SubScores) != 3 {
		t.Fatalf("expected 3 sub-scores, got %d", len(r.SubScores))
	}
	for _, ss := range r.SubScores {
		if ss.Score != 100 {
			t.Errorf("SubScore %q = %d, want 100", ss.Label, ss.Score)
		}
	}
}

func TestGoogleSpecRequiredSubScoreDecreases(t *testing.T) {
	// Product missing required "price" → Required score < 100.
	f := feedFromItem(t, `
      <g:id>prod-001</g:id>
      <g:title>Blue Running Shoes</g:title>
      <g:description>A great pair of shoes.</g:description>
      <g:availability>in stock</g:availability>
      <g:link>https://example.com/products/prod-001</g:link>
      <g:image_link>https://example.com/images/prod-001.jpg</g:image_link>
      <g:brand>RunFast</g:brand>
      <g:google_product_category>Shoes</g:google_product_category>
      <g:mpn>RUN-001-BLU</g:mpn>
      <g:additional_image_link>https://example.com/images/prod-001-side.jpg</g:additional_image_link>
      <g:product_type>Footwear &gt; Running Shoes</g:product_type>`)
	r := runChecker(t, f)
	if len(r.SubScores) == 0 {
		t.Fatal("expected sub-scores, got none")
	}
	for _, ss := range r.SubScores {
		if ss.Label == "Required" && ss.Score == 100 {
			t.Errorf("Required sub-score should be < 100 when required field missing, got 100")
		}
	}
}

func TestGoogleSpecRecommendedSubScoreDecreases(t *testing.T) {
	// Product missing recommended "brand" → Recommended score < 100.
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
	if len(r.SubScores) == 0 {
		t.Fatal("expected sub-scores, got none")
	}
	for _, ss := range r.SubScores {
		if ss.Label == "Recommended" && ss.Score == 100 {
			t.Errorf("Recommended sub-score should be < 100 when recommended field missing, got 100")
		}
	}
}

func TestGoogleSpecFormatSubScoreDecreases(t *testing.T) {
	// Product has invalid "condition" value → Format score < 100.
	f := feedFromItem(t, requiredBase+`
      <g:condition>like-new</g:condition>`)
	r := runChecker(t, f)
	if len(r.SubScores) == 0 {
		t.Fatal("expected sub-scores, got none")
	}
	for _, ss := range r.SubScores {
		if ss.Label == "Format" && ss.Score == 100 {
			t.Errorf("Format sub-score should be < 100 when format is invalid, got 100")
		}
	}
}

func TestGoogleSpecExamplesPopulated(t *testing.T) {
	// Missing required "price" → examples for that item should be non-empty.
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
	for _, item := range r.Items {
		if item.Field == "price" {
			if len(item.Examples) == 0 {
				t.Error("expected examples for price violation, got none")
			}
			return
		}
	}
	t.Error("expected item for field \"price\"")
}

func TestGoogleSpecExamplesCappedAt10(t *testing.T) {
	// 15 products all missing "price" → at most 10 examples.
	items := make([]string, 15)
	for i := range items {
		items[i] = fmt.Sprintf(`
      <g:id>prod-%03d</g:id>
      <g:title>Product %d</g:title>
      <g:description>Description for product %d.</g:description>
      <g:availability>in stock</g:availability>
      <g:link>https://example.com/products/prod-%03d</g:link>
      <g:image_link>https://example.com/images/prod-%03d.jpg</g:image_link>
      <g:brand>Brand</g:brand>
      <g:google_product_category>Shoes</g:google_product_category>`, i+1, i+1, i+1, i+1, i+1)
	}
	f := feedFromItems(t, items)
	r := runChecker(t, f)
	for _, item := range r.Items {
		if item.Field == "price" {
			if len(item.Examples) == 0 {
				t.Errorf("expected examples, got none")
			}
			if len(item.Examples) > 10 {
				t.Errorf("examples capped at 10, got %d", len(item.Examples))
			}
			return
		}
	}
	t.Error("expected item for field \"price\"")
}

func TestGoogleSpecExamplesShowMissingLabel(t *testing.T) {
	// Missing field should produce an example containing "(missing)".
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
	for _, item := range r.Items {
		if item.Field == "price" {
			for _, ex := range item.Examples {
				if strings.Contains(ex, "(missing)") {
					return
				}
			}
			t.Errorf("expected example to contain \"(missing)\", got %v", item.Examples)
			return
		}
	}
	t.Error("expected item for field \"price\"")
}

func TestGoogleSpecScoreIsWeightedAverage(t *testing.T) {
	// All required present (reqScore=100), no recommended (recScore=0), no format violations (fmtScore=100).
	// weighted = (100*6 + 0*3 + 100) / 10 = 700/10 = 70
	f := feedFromItem(t, `
      <g:id>prod-001</g:id>
      <g:title>Blue Running Shoes</g:title>
      <g:description>A great pair of running shoes for all terrain use.</g:description>
      <g:price>89.99 SEK</g:price>
      <g:availability>in stock</g:availability>
      <g:link>https://example.com/products/prod-001</g:link>
      <g:image_link>https://example.com/images/prod-001.jpg</g:image_link>`)
	r := runChecker(t, f)
	if r.Score == nil {
		t.Fatal("Score should not be nil")
	}
	if *r.Score != 70 {
		t.Errorf("Score = %d, want 70 (weighted average)", *r.Score)
	}
}

func TestGoogleSpecScoreNot100WhenMinorViolation(t *testing.T) {
	// 100 products: 99 have all fields including brand, 1 is missing brand.
	// recScore = 99, reqScore = 100, fmtScore = 100.
	// Weighted = (100*6 + 99*3 + 100) / 10 = 997/10 = 99.
	// The score must not inflate to 100 via rounding.
	const base = `
      <g:id>%s</g:id>
      <g:title>Product</g:title>
      <g:description>A great pair of running shoes for all terrain use.</g:description>
      <g:price>89.99 SEK</g:price>
      <g:availability>in stock</g:availability>
      <g:link>https://example.com/p</g:link>
      <g:image_link>https://example.com/img.jpg</g:image_link>
      <g:google_product_category>Shoes</g:google_product_category>
      <g:mpn>RUN-001</g:mpn>
      <g:additional_image_link>https://example.com/img2.jpg</g:additional_image_link>
      <g:product_type>Footwear</g:product_type>`
	items := make([]string, 100)
	for i := range items {
		id := fmt.Sprintf("p%03d", i)
		if i == 0 {
			items[i] = fmt.Sprintf(base, id) // no brand → recommended violation
		} else {
			items[i] = fmt.Sprintf(base+"\n      <g:brand>B</g:brand>", id)
		}
	}
	f := feedFromItems(t, items)
	r := runChecker(t, f)
	if r.Score == nil {
		t.Fatal("Score should not be nil")
	}
	if *r.Score >= 100 {
		t.Errorf("Score = %d, want < 100 when violations exist (minor violation must not inflate to 100)", *r.Score)
	}
}

func TestGoogleSpecExamplesShowBadValue(t *testing.T) {
	// Invalid format value should appear in the example string.
	f := feedFromItem(t, requiredBase+`
      <g:condition>like-new</g:condition>`)
	r := runChecker(t, f)
	for _, item := range r.Items {
		if item.Field == "condition" {
			for _, ex := range item.Examples {
				if strings.Contains(ex, "like-new") {
					return
				}
			}
			t.Errorf("expected example to contain bad value \"like-new\", got %v", item.Examples)
			return
		}
	}
	t.Error("expected item for field \"condition\"")
}

func TestGoogleSpecAffectedProductsPopulated(t *testing.T) {
	// Two products: one missing brand, one has brand.
	raw := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:g="http://base.google.com/ns/1.0">
  <channel>
    <item>
      <g:id>sku-001</g:id><g:title>No Brand</g:title>
      <g:description>D</g:description><g:price>10 SEK</g:price>
      <g:availability>in stock</g:availability>
      <g:link>https://x.com</g:link><g:image_link>https://x.com/img.jpg</g:image_link>
    </item>
    <item>
      <g:id>sku-002</g:id><g:title>Has Brand</g:title>
      <g:description>D</g:description><g:price>10 SEK</g:price>
      <g:availability>in stock</g:availability>
      <g:link>https://x.com</g:link><g:image_link>https://x.com/img.jpg</g:image_link>
      <g:brand>Acme</g:brand>
    </item>
  </channel>
</rss>`)
	f, err := feed.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	r := googlespec.NewChecker().Run(context.Background(), f)

	for _, item := range r.Items {
		if item.Field == "brand" {
			if len(item.AffectedProducts) != 1 {
				t.Fatalf("AffectedProducts len = %d, want 1", len(item.AffectedProducts))
			}
			if item.AffectedProducts[0].ID != "sku-001" {
				t.Errorf("AffectedProducts[0].ID = %q, want sku-001", item.AffectedProducts[0].ID)
			}
			// Examples still capped at 10; AffectedProducts is the full list (no cap).
			if len(item.Examples) > 10 {
				t.Errorf("Examples len = %d, want <= 10", len(item.Examples))
			}
			return
		}
	}
	t.Error("expected item for field 'brand'")
}

func TestGoogleSpecAffectedProductsNotCapped(t *testing.T) {
	// 15 products all missing brand — AffectedProducts should have 15, Examples capped at 10.
	const count = 15
	items := ""
	for i := range count {
		items += fmt.Sprintf(`<item>
      <g:id>sku-%03d</g:id><g:title>Prod %d</g:title>
      <g:description>D</g:description><g:price>10 SEK</g:price>
      <g:availability>in stock</g:availability>
      <g:link>https://x.com</g:link><g:image_link>https://x.com/img.jpg</g:image_link>
    </item>`, i, i)
	}
	raw := []byte(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:g="http://base.google.com/ns/1.0"><channel>` + items + `</channel></rss>`)
	f, err := feed.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	r := googlespec.NewChecker().Run(context.Background(), f)

	for _, item := range r.Items {
		if item.Field == "brand" {
			if len(item.AffectedProducts) != count {
				t.Errorf("AffectedProducts len = %d, want %d", len(item.AffectedProducts), count)
			}
			if len(item.Examples) > 10 {
				t.Errorf("Examples len = %d, want <= 10", len(item.Examples))
			}
			return
		}
	}
	t.Error("expected item for field 'brand'")
}
