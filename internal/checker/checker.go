// Package checker defines the Checker interface and result types used by all feed checkers.
package checker

import (
	"context"

	"github.com/johlun99/merchant-analyzer/internal/feed"
)

// Status represents the outcome of a checker run.
type Status int

// Status values represent the outcome severity of a checker run.
const (
	StatusOK      Status = iota // no issues found
	StatusWarning               // found issues but not critical
	StatusError                 // found significant issues
	StatusFatal                 // checker could not run
)

// SubScore is a named score component (0–100).
type SubScore struct {
	Label string
	Score int
}

// AffectedProduct holds the ID, Title, and optional per-product context for a checker finding.
type AffectedProduct struct {
	ID    string
	Title string
	Value string // optional: per-product context, e.g. "color: (missing)" or "description: 45 chars"
}

// Item is a single finding from a checker.
type Item struct {
	Field            string
	Message          string
	Count            int               // number of products affected
	Examples         []string          // up to 10 affected product descriptions; nil for non-googlespec checkers
	AffectedProducts []AffectedProduct // full list of affected products, no cap — for export and TUI drill-down
	Impact           string            // optional: "High", "Medium", or "Low"
	ImpactDesc       string            // optional: one-liner explaining why this attribute matters for AI
}

// Result is the outcome of running a single checker.
type Result struct {
	Name      string
	Status    Status
	Items     []Item
	Score     *int       // optional 0–100 score
	SubScores []SubScore // optional named sub-scores; for Google Spec: Required, Recommended, Format
	Err       error      // set when Status == StatusFatal
}

// Checker analyzes a feed and returns a Result.
type Checker interface {
	Name() string
	Run(ctx context.Context, f *feed.Feed) Result
}

// ScoreOf is a helper that returns a pointer to an int score.
func ScoreOf(n int) *int {
	return &n
}
