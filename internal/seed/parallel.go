package seed

import (
	"context"
	"sync"
)

// parallel runs fn over items with at most `concurrency` goroutines in flight.
// It stops scheduling new work after the first error (cancelling the context
// passed to in-flight fn calls) and returns that error. A concurrency < 1 is
// treated as 1.
func parallel[T any](
	ctx context.Context, items []T, concurrency int, fn func(context.Context, T) error,
) error {
	if concurrency < 1 {
		concurrency = 1
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		sem      = make(chan struct{}, concurrency)
		wg       sync.WaitGroup
		once     sync.Once
		firstErr error
	)

	for _, it := range items {
		if ctx.Err() != nil {
			break
		}
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		go func(it T) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fn(ctx, it); err != nil {
				once.Do(func() {
					firstErr = err
					cancel()
				})
			}
		}(it)
	}

	wg.Wait()
	return firstErr
}
