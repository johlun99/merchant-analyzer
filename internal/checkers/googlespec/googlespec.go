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
//
//nolint:cyclop
func (c *Checker) Run(_ context.Context, f *feed.Feed) checker.Result {
	if len(f.Products) == 0 {
		return checker.Result{Name: c.Name(), Status: checker.StatusOK}
	}

	total := len(f.Products)
	requiredFailed := make([]bool, total)
	recFailed := make([]bool, total)
	fmtFailed := make([]bool, total)

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

		markFailures(f.Products, rule, requiredFailed, recFailed, fmtFailed)

		items = append(items, checker.Item{
			Field:    rule.field,
			Message:  buildMessage(rule, presence, format, total),
			Count:    presence + format,
			Examples: collectExamples(f.Products, rule, 10),
		})
	}

	reqScore := scoreFromFailed(requiredFailed)
	recScore := scoreFromFailed(recFailed)
	fmtScore := scoreFromFailed(fmtFailed)

	return checker.Result{
		Name:   c.Name(),
		Status: status,
		Items:  items,
		Score:  checker.ScoreOf((reqScore*6 + recScore*3 + fmtScore + 5) / 10),
		SubScores: []checker.SubScore{
			{Label: "Required", Score: reqScore},
			{Label: "Recommended", Score: recScore},
			{Label: "Format", Score: fmtScore},
		},
	}
}

// markFailures records which products violated rule into the appropriate category slice.
func markFailures(products []feed.Product, rule fieldRule, reqFailed, recFailed, fmtFailed []bool) {
	for i, p := range products {
		if !isViolation(rule, &p) {
			continue
		}
		switch {
		case rule.required:
			reqFailed[i] = true
		case rule.checkPresence:
			recFailed[i] = true
		default:
			fmtFailed[i] = true
		}
	}
}

// isViolation returns true if the product violates the rule.
func isViolation(rule fieldRule, p *feed.Product) bool {
	v := rule.get(p)
	if rule.checkPresence && v == "" {
		return true
	}
	return rule.validate != nil && v != "" && !rule.validate(v)
}

// scoreFromFailed returns a 0–100 score: 100 × compliant / total.
func scoreFromFailed(failed []bool) int {
	if len(failed) == 0 {
		return 100
	}
	compliant := 0
	for _, f := range failed {
		if !f {
			compliant++
		}
	}
	return compliant * 100 / len(failed)
}

// collectExamples returns up to max example strings for products that violate rule.
// Each example has the format: `id "Title" — field: value` (or `(missing)` if absent).
func collectExamples(products []feed.Product, rule fieldRule, limit int) []string {
	var examples []string
	for _, p := range products {
		if len(examples) >= limit {
			break
		}
		v := rule.get(&p)
		if rule.checkPresence && v == "" {
			examples = append(examples, fmt.Sprintf("%s %q — %s: (missing)", p.ID, p.Title, rule.field))
			continue
		}
		if rule.validate != nil && v != "" && !rule.validate(v) {
			examples = append(examples, fmt.Sprintf("%s %q — %s: %s", p.ID, p.Title, rule.field, v))
		}
	}
	return examples
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
