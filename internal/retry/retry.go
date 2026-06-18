package retry

import (
	"context"
	"time"
)

func Do(ctx context.Context, attempts int, initialDelay time.Duration, fn func() error) error {
	if attempts < 1 {
		attempts = 1
	}

	delay := initialDelay
	if delay <= 0 {
		delay = 100 * time.Millisecond
	}

	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		if attempt == attempts {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		delay *= 2
		if delay > 5*time.Second {
			delay = 5 * time.Second
		}
	}

	return err
}
