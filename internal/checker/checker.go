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

// Item is a single finding from a checker.
type Item struct {
	Field    string
	Message  string
	Count    int      // number of products affected
	Examples []string // up to 10 affected product descriptions; nil for non-googlespec checkers
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
