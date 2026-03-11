// Package attributes checks required Google Shopping attribute coverage.
package attributes

import (
	"context"
	"fmt"

	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/feed"
)

// requiredFields are the Google Shopping fields checked for coverage.
var requiredFields = []string{
	"id", "title", "description", "price", "availability", "link", "image_link",
}

// Checker measures coverage of required Google Shopping attributes.
type Checker struct{}

// NewChecker returns a new attribute Checker.
func NewChecker() *Checker {
	return &Checker{}
}

// Name returns the checker name.
func (c *Checker) Name() string {
	return "Attribute Check"
}

// Run checks what percentage of products are missing each required field.
func (c *Checker) Run(_ context.Context, f *feed.Feed) checker.Result {
	if len(f.Products) == 0 {
		return checker.Result{Name: c.Name(), Status: checker.StatusOK}
	}

	total := len(f.Products)
	var items []checker.Item
	status := checker.StatusOK

	for _, field := range requiredFields {
		missing := countMissing(f.Products, field)
		if missing == 0 {
			continue
		}

		pct := float64(missing) / float64(total) * 100
		var fieldStatus checker.Status
		switch {
		case pct >= 5:
			fieldStatus = checker.StatusError
		default:
			fieldStatus = checker.StatusWarning
		}

		if fieldStatus > status {
			status = fieldStatus
		}

		items = append(items, checker.Item{
			Field:   field,
			Message: fmt.Sprintf("%.1f%% of products missing %q (%d of %d)", pct, field, missing, total),
			Count:   missing,
		})
	}

	return checker.Result{
		Name:   c.Name(),
		Status: status,
		Items:  items,
	}
}

// countMissing returns how many products have an empty value for the given field.
func countMissing(products []feed.Product, field string) int {
	count := 0
	for _, p := range products {
		if getField(&p, field) == "" {
			count++
		}
	}
	return count
}

// getField returns the value of a named required field from a product.
func getField(p *feed.Product, field string) string {
	switch field {
	case "id":
		return p.ID
	case "title":
		return p.Title
	case "description":
		return p.Description
	case "price":
		return p.Price
	case "availability":
		return p.Availability
	case "link":
		return p.Link
	case "image_link":
		return p.ImageLink
	default:
		return ""
	}
}
