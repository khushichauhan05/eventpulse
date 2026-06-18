package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/apekshita/eventpulse/internal/retry"
)

var errTemp = errors.New("temporary error")

func TestDo_SucceedsOnFirstAttempt(t *testing.T) {
	calls := 0
	err := retry.Do(context.Background(), 3, time.Millisecond, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDo_RetriesUntilSuccess(t *testing.T) {
	calls := 0
	err := retry.Do(context.Background(), 5, time.Millisecond, func() error {
		calls++
		if calls < 3 {
			return errTemp
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil after retries, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDo_ExhaustsAttempts(t *testing.T) {
	calls := 0
	err := retry.Do(context.Background(), 3, time.Millisecond, func() error {
		calls++
		return errTemp
	})
	if !errors.Is(err, errTemp) {
		t.Fatalf("expected errTemp, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected exactly 3 calls, got %d", calls)
	}
}

func TestDo_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	err := retry.Do(ctx, 10, time.Millisecond, func() error {
		calls++
		return errTemp
	})
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
	// Only the first attempt fires before the context is checked between retries.
	if calls > 2 {
		t.Fatalf("too many calls on cancelled context: %d", calls)
	}
}

func TestDo_ZeroAttemptsBecomesOne(t *testing.T) {
	calls := 0
	_ = retry.Do(context.Background(), 0, time.Millisecond, func() error {
		calls++
		return errTemp
	})
	if calls != 1 {
		t.Fatalf("expected 1 call for zero attempts, got %d", calls)
	}
}
