// Package analyzer runs feed checkers concurrently and collects results.
package analyzer

import (
	"context"
	"sync"

	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/feed"
)

// Run executes all checkers concurrently and calls send for each result.
// send is called from goroutines; the caller must handle thread safety if needed.
// Cancelling ctx propagates to all in-flight checkers via their Run method.
func Run(ctx context.Context, f *feed.Feed, checkers []checker.Checker, send func(checker.Result)) {
	var wg sync.WaitGroup
	for _, c := range checkers {
		wg.Add(1)
		go func(c checker.Checker) {
			defer wg.Done()
			result := c.Run(ctx, f)
			send(result)
		}(c)
	}
	wg.Wait()
}
