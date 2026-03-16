// Package aireadiness scores feed readiness for AI-powered product discovery.
package aireadiness

import (
	"context"
	"fmt"
	"math"

	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/feed"
)

const minDescriptionLen = 150

// fieldImpact holds static AI-readiness impact metadata for a feed attribute.
type fieldImpact struct {
	level string // "High", "Medium", or "Low"
	desc  string // one-liner explaining why this attribute matters for AI
}

//nolint:gochecknoglobals
var impactRegistry = map[string]fieldImpact{
	// UCP fields
	"id":           {"High", "Unique IDs let AI systems track and de-duplicate products reliably"},
	"title":        {"High", "Titles are the primary signal for semantic search and discovery"},
	"description":  {"High", "Rich descriptions are the main LLM context for product understanding"},
	"price":        {"Medium", "Price data is required for transaction-eligible search results"},
	"availability": {"Medium", "Availability filters keep AI results shoppable in real time"},
	"link":         {"Low", "Product URLs are required for click-through but rarely affect AI ranking"},
	"image_link":   {"High", "Primary images power visual AI search and product recognition"},
	"brand":        {"High", "Brand signals sharpen recommendation precision and semantic matching"},
	"gtin_or_mpn":  {"High", "Unique identifiers enable AI to match products across catalogs"},
	"condition":    {"Medium", "Condition signals refine AI results for new vs. used product queries"},
	// LLM attributes
	"color":              {"High", "Color attributes are critical for visual AI discovery and filtering"},
	"size":               {"High", "Size data powers personalised fit recommendations"},
	"material":           {"Medium", "Material details enrich semantic product context for LLMs"},
	"description_length": {"High", "Longer descriptions give LLMs richer context for intent matching"},
	// Image attributes
	"additional_image_link": {"High", "Multiple angles significantly improve visual AI recognition accuracy"},
}

// Checker evaluates how well a feed is optimized for AI-powered product discovery.
type Checker struct{}

// NewChecker returns a new AI readiness Checker.
func NewChecker() *Checker {
	return &Checker{}
}

// Name returns the checker name.
func (c *Checker) Name() string {
	return "AI Readiness"
}

// Run scores the feed's AI readiness from 0 to 100.
//
// Scoring formula:
//   - UCP compliance (max 40 pts): coverage of 10 UCP fields averaged, × 40
//   - LLM attributes (max 35 pts): coverage of 5 LLM checks averaged, × 35
//   - Image quality (max 25 pts): image_link × 15 pts + additional_image_link × 10 pts
func (c *Checker) Run(_ context.Context, f *feed.Feed) checker.Result {
	if len(f.Products) == 0 {
		score := 0
		return checker.Result{
			Name:   c.Name(),
			Status: checker.StatusError,
			Score:  &score,
			Items:  []checker.Item{{Field: "products", Message: "feed has no products"}},
		}
	}

	ucpScore, ucpItems := computeUCPScore(f.Products)
	llmScore, llmItems := computeLLMScore(f.Products)
	imgScore, imgItems := computeImageScore(f.Products)

	total := int(math.Round(ucpScore + llmScore + imgScore))
	if total > 100 {
		total = 100
	}

	var items []checker.Item
	items = append(items, ucpItems...)
	items = append(items, llmItems...)
	items = append(items, imgItems...)

	status := scoreToStatus(total)

	return checker.Result{
		Name:   c.Name(),
		Status: status,
		Items:  items,
		Score:  checker.ScoreOf(total),
	}
}

// impactFor returns the Impact and ImpactDesc for a field name from the registry.
func impactFor(field string) (level, desc string) {
	if meta, ok := impactRegistry[field]; ok {
		return meta.level, meta.desc
	}
	return "", ""
}

// collectAIExamples returns up to limit example strings for products where failing returns true.
// label(p) produces the per-product description suffix, e.g. "color: (missing)".
func collectAIExamples(products []feed.Product, failing func(*feed.Product) bool, label func(*feed.Product) string, limit int) []string {
	var examples []string
	for i := range products {
		if len(examples) >= limit {
			break
		}
		p := &products[i]
		if failing(p) {
			examples = append(examples, fmt.Sprintf("%s %q — %s", p.ID, p.Title, label(p)))
		}
	}
	return examples
}

// collectAIAffected returns all products where failing returns true as AffectedProduct entries (no cap).
// label(p) produces the per-product context string stored in Value (e.g. "color: (missing)").
func collectAIAffected(products []feed.Product, failing func(*feed.Product) bool, label func(*feed.Product) string) []checker.AffectedProduct {
	var affected []checker.AffectedProduct
	for i := range products {
		p := &products[i]
		if failing(p) {
			affected = append(affected, checker.AffectedProduct{ID: p.ID, Title: p.Title, Value: label(p)})
		}
	}
	return affected
}

// computeUCPScore returns the UCP sub-score (max 40) and findings.
func computeUCPScore(products []feed.Product) (float64, []checker.Item) {
	ucpFields := []struct {
		name string
		get  func(*feed.Product) string
	}{
		{"id", func(p *feed.Product) string { return p.ID }},
		{"title", func(p *feed.Product) string { return p.Title }},
		{"description", func(p *feed.Product) string { return p.Description }},
		{"price", func(p *feed.Product) string { return p.Price }},
		{"availability", func(p *feed.Product) string { return p.Availability }},
		{"link", func(p *feed.Product) string { return p.Link }},
		{"image_link", func(p *feed.Product) string { return p.ImageLink }},
		{"brand", func(p *feed.Product) string { return p.Brand }},
		{"gtin_or_mpn", func(p *feed.Product) string {
			if p.GTIN != "" {
				return p.GTIN
			}
			return p.MPN
		}},
		{"condition", func(p *feed.Product) string { return p.Condition }},
	}

	total := len(products)
	var items []checker.Item
	sumCoverage := 0.0

	for _, f := range ucpFields {
		present := 0
		for i := range products {
			if f.get(&products[i]) != "" {
				present++
			}
		}
		coverage := float64(present) / float64(total)
		sumCoverage += coverage
		if coverage < 1.0 {
			missing := total - present
			fieldName := f.name
			failing := func(p *feed.Product) bool { return f.get(p) == "" }
			impLvl, impDesc := impactFor(fieldName)
			ucpLabel := func(_ *feed.Product) string { return fieldName + ": (missing)" }
			items = append(items, checker.Item{
				Field:            fieldName,
				Message:          fmt.Sprintf("UCP: %d of %d products missing %q", missing, total, fieldName),
				Count:            missing,
				Examples:         collectAIExamples(products, failing, ucpLabel, 10),
				AffectedProducts: collectAIAffected(products, failing, ucpLabel),
				Impact:           impLvl,
				ImpactDesc:       impDesc,
			})
		}
	}

	score := (sumCoverage / float64(len(ucpFields))) * 40
	return score, items
}

// computeLLMScore returns the LLM sub-score (max 35) and findings.
func computeLLMScore(products []feed.Product) (float64, []checker.Item) {
	type llmCheck struct {
		name  string
		pass  func(*feed.Product) bool
		label func(*feed.Product) string
	}

	checks := []llmCheck{
		{"color",
			func(p *feed.Product) bool { return p.Color != "" },
			func(_ *feed.Product) string { return "color: (missing)" }},
		{"size",
			func(p *feed.Product) bool { return p.Size != "" },
			func(_ *feed.Product) string { return "size: (missing)" }},
		{"material",
			func(p *feed.Product) bool { return p.Material != "" },
			func(_ *feed.Product) string { return "material: (missing)" }},
		{"description_length",
			func(p *feed.Product) bool { return len(p.Description) > minDescriptionLen },
			func(p *feed.Product) string {
				return fmt.Sprintf("description: %d chars (need >%d)", len(p.Description), minDescriptionLen)
			}},
	}

	total := len(products)
	var items []checker.Item
	sumCoverage := 0.0

	for _, ch := range checks {
		passing := 0
		for i := range products {
			if ch.pass(&products[i]) {
				passing++
			}
		}
		coverage := float64(passing) / float64(total)
		sumCoverage += coverage
		if coverage < 1.0 {
			missing := total - passing
			failing := func(p *feed.Product) bool { return !ch.pass(p) }
			impLvl, impDesc := impactFor(ch.name)
			items = append(items, checker.Item{
				Field:            ch.name,
				Message:          fmt.Sprintf("LLM: %d of %d products missing %q", missing, total, ch.name),
				Count:            missing,
				Examples:         collectAIExamples(products, failing, ch.label, 10),
				AffectedProducts: collectAIAffected(products, failing, ch.label),
				Impact:           impLvl,
				ImpactDesc:       impDesc,
			})
		}
	}

	score := (sumCoverage / float64(len(checks))) * 35
	return score, items
}

// computeImageScore returns the image sub-score (max 25) and findings.
func computeImageScore(products []feed.Product) (float64, []checker.Item) {
	total := len(products)
	withImage := 0
	withAdditional := 0

	for _, p := range products {
		if p.ImageLink != "" {
			withImage++
		}
		if len(p.AdditionalImages) > 0 {
			withAdditional++
		}
	}

	imageCoverage := float64(withImage) / float64(total)
	additionalCoverage := float64(withAdditional) / float64(total)
	score := imageCoverage*15 + additionalCoverage*10

	var items []checker.Item
	if imageCoverage < 1.0 {
		missing := total - withImage
		failing := func(p *feed.Product) bool { return p.ImageLink == "" }
		imgLabel := func(_ *feed.Product) string { return "image_link: (missing)" }
		impLvl, impDesc := impactFor("image_link")
		items = append(items, checker.Item{
			Field:            "image_link",
			Message:          fmt.Sprintf("Image: %d of %d products missing image_link", missing, total),
			Count:            missing,
			Examples:         collectAIExamples(products, failing, imgLabel, 10),
			AffectedProducts: collectAIAffected(products, failing, imgLabel),
			Impact:           impLvl,
			ImpactDesc:       impDesc,
		})
	}
	if additionalCoverage < 1.0 {
		missing := total - withAdditional
		failing := func(p *feed.Product) bool { return len(p.AdditionalImages) == 0 }
		addImgLabel := func(_ *feed.Product) string { return "additional_image_link: (missing)" }
		impLvl, impDesc := impactFor("additional_image_link")
		items = append(items, checker.Item{
			Field:            "additional_image_link",
			Message:          fmt.Sprintf("Image: %d of %d products missing additional_image_link", missing, total),
			Count:            missing,
			Examples:         collectAIExamples(products, failing, addImgLabel, 10),
			AffectedProducts: collectAIAffected(products, failing, addImgLabel),
			Impact:           impLvl,
			ImpactDesc:       impDesc,
		})
	}

	return score, items
}

// scoreToStatus converts a 0–100 score to a Status.
func scoreToStatus(score int) checker.Status {
	switch {
	case score >= 80:
		return checker.StatusOK
	case score >= 50:
		return checker.StatusWarning
	default:
		return checker.StatusError
	}
}
