package analyzer_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/johlun99/merchant-analyzer/internal/analyzer"
	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/feed"
)

// stubChecker is a test double that returns a preset result after an optional delay.
type stubChecker struct {
	name   string
	result checker.Result
	delay  time.Duration
}

func (s *stubChecker) Name() string { return s.name }
func (s *stubChecker) Run(_ context.Context, _ *feed.Feed) checker.Result {
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	return s.result
}

func TestRunCollectsAllResults(t *testing.T) {
	checkers := []checker.Checker{
		&stubChecker{name: "A", result: checker.Result{Name: "A", Status: checker.StatusOK}},
		&stubChecker{name: "B", result: checker.Result{Name: "B", Status: checker.StatusWarning}},
		&stubChecker{name: "C", result: checker.Result{Name: "C", Status: checker.StatusError}},
	}

	var mu sync.Mutex
	var results []checker.Result
	send := func(r checker.Result) {
		mu.Lock()
		results = append(results, r)
		mu.Unlock()
	}

	f := &feed.Feed{}
	analyzer.Run(context.Background(), f, checkers, send)

	if len(results) != 3 {
		t.Errorf("got %d results, want 3", len(results))
	}
}

func TestRunExecutesConcurrently(t *testing.T) {
	delay := 50 * time.Millisecond
	checkers := []checker.Checker{
		&stubChecker{name: "A", delay: delay},
		&stubChecker{name: "B", delay: delay},
		&stubChecker{name: "C", delay: delay},
	}

	var mu sync.Mutex
	var results []checker.Result
	send := func(r checker.Result) {
		mu.Lock()
		results = append(results, r)
		mu.Unlock()
	}

	start := time.Now()
	analyzer.Run(context.Background(), &feed.Feed{}, checkers, send)
	elapsed := time.Since(start)

	// Sequential would take 3×delay; concurrent should finish in ~1×delay
	if elapsed >= 3*delay {
		t.Errorf("elapsed %v >= sequential time %v — checkers are not running concurrently", elapsed, 3*delay)
	}
	if len(results) != 3 {
		t.Errorf("got %d results, want 3", len(results))
	}
}

func TestRunRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	called := false
	checkers := []checker.Checker{
		&stubChecker{name: "A", result: checker.Result{Name: "A"}},
		// A real checker would check ctx.Done(); our stub doesn't, so we just
		// verify Run returns after all checkers complete even if ctx is done.
	}

	analyzer.Run(ctx, &feed.Feed{}, checkers, func(_ checker.Result) {
		called = true
	})

	// Run should have completed (not hung) even with a cancelled context
	if !called {
		t.Error("send callback was never called")
	}
}

func TestRunWithNoCheckers(t *testing.T) {
	var results []checker.Result
	analyzer.Run(context.Background(), &feed.Feed{}, nil, func(r checker.Result) {
		results = append(results, r)
	})
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}
