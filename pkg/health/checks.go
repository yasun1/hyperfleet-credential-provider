package health

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPCheck creates a health check that verifies an HTTP endpoint is reachable
func HTTPCheck(url string, timeout time.Duration) Check {
	return func(ctx context.Context) error {
		client := &http.Client{
			Timeout: timeout,
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		return nil
	}
}

// AlwaysHealthy is a check that always returns healthy
// Useful for testing or default checks
func AlwaysHealthy() Check {
	return func(ctx context.Context) error {
		return nil
	}
}

// AlwaysUnhealthy is a check that always returns unhealthy
// Useful for testing
func AlwaysUnhealthy(reason string) Check {
	return func(ctx context.Context) error {
		return fmt.Errorf("%s", reason)
	}
}

// TimeBasedCheck creates a check that is only healthy during a time window
// Useful for maintenance windows
func TimeBasedCheck(startHour, endHour int) Check {
	return func(ctx context.Context) error {
		now := time.Now()
		currentHour := now.Hour()

		if currentHour >= startHour && currentHour < endHour {
			return nil
		}

		return fmt.Errorf("outside of healthy time window (%d:00 - %d:00)", startHour, endHour)
	}
}

// CombinedCheck combines multiple checks with AND logic
// All checks must pass for the combined check to pass
func CombinedCheck(checks ...Check) Check {
	return func(ctx context.Context) error {
		for i, check := range checks {
			if err := check(ctx); err != nil {
				return fmt.Errorf("check %d failed: %w", i, err)
			}
		}
		return nil
	}
}

// AnyCheck combines multiple checks with OR logic
// At least one check must pass for the combined check to pass
func AnyCheck(checks ...Check) Check {
	return func(ctx context.Context) error {
		var lastErr error
		for _, check := range checks {
			if err := check(ctx); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
		if lastErr != nil {
			return fmt.Errorf("all checks failed, last error: %w", lastErr)
		}
		return fmt.Errorf("all checks failed")
	}
}

// DelayedCheck creates a check that only becomes healthy after a delay
// Useful for startup grace periods
func DelayedCheck(delay time.Duration, underlying Check) Check {
	startTime := time.Now()
	return func(ctx context.Context) error {
		if time.Since(startTime) < delay {
			return fmt.Errorf("still in grace period (%.0fs remaining)", (delay - time.Since(startTime)).Seconds())
		}
		return underlying(ctx)
	}
}

// ContextCheck wraps a function that takes a context
func ContextCheck(fn func(context.Context) error) Check {
	return fn
}

// SimpleCheck wraps a function that doesn't need context
func SimpleCheck(fn func() error) Check {
	return func(ctx context.Context) error {
		return fn()
	}
}
