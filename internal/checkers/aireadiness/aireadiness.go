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
			items = append(items, checker.Item{
				Field:   f.name,
				Message: fmt.Sprintf("UCP: %d of %d products missing %q", missing, total, f.name),
				Count:   missing,
			})
		}
	}

	score := (sumCoverage / float64(len(ucpFields))) * 40
	return score, items
}

// computeLLMScore returns the LLM sub-score (max 35) and findings.
func computeLLMScore(products []feed.Product) (float64, []checker.Item) {
	type llmCheck struct {
		name string
		pass func(*feed.Product) bool
	}

	checks := []llmCheck{
		{"color", func(p *feed.Product) bool { return p.Color != "" }},
		{"size", func(p *feed.Product) bool { return p.Size != "" }},
		{"material", func(p *feed.Product) bool { return p.Material != "" }},
		{"additional_image_link", func(p *feed.Product) bool { return len(p.AdditionalImages) > 0 }},
		{"description_length", func(p *feed.Product) bool { return len(p.Description) > minDescriptionLen }},
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
			items = append(items, checker.Item{
				Field:   ch.name,
				Message: fmt.Sprintf("LLM: %d of %d products missing %q", missing, total, ch.name),
				Count:   missing,
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
		items = append(items, checker.Item{
			Field:   "image_link",
			Message: fmt.Sprintf("Image: %d of %d products missing image_link", missing, total),
			Count:   missing,
		})
	}
	if additionalCoverage < 1.0 {
		missing := total - withAdditional
		items = append(items, checker.Item{
			Field:   "additional_image_link",
			Message: fmt.Sprintf("Image: %d of %d products missing additional_image_link", missing, total),
			Count:   missing,
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
