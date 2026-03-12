// Package googlespec validates Google Merchant Center feed attribute presence and format.
package googlespec

import (
	"context"
	"fmt"

	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/feed"
)

// Checker validates feed attributes against the Google Merchant Center spec.
type Checker struct{}

// NewChecker returns a new Google Feed Spec Checker.
func NewChecker() *Checker { return &Checker{} }

// Name returns the checker name.
func (c *Checker) Name() string { return "Google Feed Spec" }

// Run validates all products against Google Merchant Center attribute rules.
func (c *Checker) Run(_ context.Context, f *feed.Feed) checker.Result {
	if len(f.Products) == 0 {
		return checker.Result{Name: c.Name(), Status: checker.StatusOK}
	}

	total := len(f.Products)
	var items []checker.Item
	status := checker.StatusOK

	for _, rule := range rules {
		presence, format := countViolations(f.Products, rule)
		if presence+format == 0 {
			continue
		}

		ruleStatus := checker.StatusWarning
		if rule.required {
			ruleStatus = checker.StatusError
		}
		if ruleStatus > status {
			status = ruleStatus
		}

		items = append(items, checker.Item{
			Field:   rule.field,
			Message: buildMessage(rule, presence, format, total),
			Count:   presence + format,
		})
	}

	return checker.Result{
		Name:   c.Name(),
		Status: status,
		Items:  items,
	}
}

// countViolations returns separate counts for presence violations (empty value)
// and format violations (non-empty value that fails validate).
func countViolations(products []feed.Product, rule fieldRule) (presenceCount, formatCount int) {
	for _, p := range products {
		v := rule.get(&p)
		if rule.checkPresence && v == "" {
			presenceCount++
			continue
		}
		if rule.validate != nil && v != "" && !rule.validate(v) {
			formatCount++
		}
	}
	return
}

// buildMessage produces a human-readable message for a violated rule.
// Presence violations and format violations use distinct message templates.
func buildMessage(rule fieldRule, presence, format, total int) string {
	violations := presence + format
	if format == 0 {
		if rule.required {
			return fmt.Sprintf("%d of %d products missing required field %q", violations, total, rule.field)
		}
		return fmt.Sprintf("%d of %d products missing recommended field %q", violations, total, rule.field)
	}
	if presence == 0 {
		return fmt.Sprintf("%d of %d products have invalid %q (expected: %s)", violations, total, rule.field, rule.hint)
	}
	// Mixed: some products are missing the field, others have bad format
	return fmt.Sprintf("%d of %d products have missing or invalid %q (expected: %s)", violations, total, rule.field, rule.hint)
}
