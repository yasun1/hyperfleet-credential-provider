package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlwaysHealthy(t *testing.T) {
	check := AlwaysHealthy()
	err := check(context.Background())
	assert.NoError(t, err)
}

func TestAlwaysUnhealthy(t *testing.T) {
	reason := "test failure reason"
	check := AlwaysUnhealthy(reason)
	err := check(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), reason)
}

func TestHTTPCheck_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	check := HTTPCheck(server.URL, 1*time.Second)
	err := check(context.Background())
	assert.NoError(t, err)
}

func TestHTTPCheck_Failure(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	check := HTTPCheck(server.URL, 1*time.Second)
	err := check(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code")
}

func TestHTTPCheck_Timeout(t *testing.T) {
	// Create test server that delays
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	check := HTTPCheck(server.URL, 100*time.Millisecond)
	err := check(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
}

func TestHTTPCheck_InvalidURL(t *testing.T) {
	check := HTTPCheck("http://localhost:99999", 1*time.Second)
	err := check(context.Background())
	require.Error(t, err)
}

func TestTimeBasedCheck(t *testing.T) {
	now := time.Now()
	currentHour := now.Hour()

	tests := []struct {
		name      string
		startHour int
		endHour   int
		wantErr   bool
	}{
		{
			name:      "within window",
			startHour: currentHour,
			endHour:   currentHour + 1,
			wantErr:   false,
		},
		{
			name:      "outside window",
			startHour: (currentHour + 2) % 24,
			endHour:   (currentHour + 3) % 24,
			wantErr:   true,
		},
		{
			name:      "all day window",
			startHour: 0,
			endHour:   24,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := TimeBasedCheck(tt.startHour, tt.endHour)
			err := check(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCombinedCheck_AllHealthy(t *testing.T) {
	check := CombinedCheck(
		AlwaysHealthy(),
		AlwaysHealthy(),
		AlwaysHealthy(),
	)

	err := check(context.Background())
	assert.NoError(t, err)
}

func TestCombinedCheck_OneUnhealthy(t *testing.T) {
	check := CombinedCheck(
		AlwaysHealthy(),
		AlwaysUnhealthy("test failure"),
		AlwaysHealthy(),
	)

	err := check(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check 1 failed")
	assert.Contains(t, err.Error(), "test failure")
}

func TestCombinedCheck_Empty(t *testing.T) {
	check := CombinedCheck()
	err := check(context.Background())
	assert.NoError(t, err)
}

func TestAnyCheck_AllHealthy(t *testing.T) {
	check := AnyCheck(
		AlwaysHealthy(),
		AlwaysHealthy(),
		AlwaysHealthy(),
	)

	err := check(context.Background())
	assert.NoError(t, err)
}

func TestAnyCheck_OneHealthy(t *testing.T) {
	check := AnyCheck(
		AlwaysUnhealthy("failure 1"),
		AlwaysHealthy(),
		AlwaysUnhealthy("failure 2"),
	)

	err := check(context.Background())
	assert.NoError(t, err)
}

func TestAnyCheck_AllUnhealthy(t *testing.T) {
	check := AnyCheck(
		AlwaysUnhealthy("failure 1"),
		AlwaysUnhealthy("failure 2"),
		AlwaysUnhealthy("failure 3"),
	)

	err := check(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all checks failed")
}

func TestAnyCheck_Empty(t *testing.T) {
	check := AnyCheck()
	err := check(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all checks failed")
}

func TestDelayedCheck_BeforeDelay(t *testing.T) {
	check := DelayedCheck(1*time.Second, AlwaysHealthy())
	err := check(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grace period")
}

func TestDelayedCheck_AfterDelay(t *testing.T) {
	check := DelayedCheck(10*time.Millisecond, AlwaysHealthy())
	time.Sleep(20 * time.Millisecond)
	err := check(context.Background())
	assert.NoError(t, err)
}

func TestDelayedCheck_UnderlyingFails(t *testing.T) {
	check := DelayedCheck(10*time.Millisecond, AlwaysUnhealthy("underlying failure"))
	time.Sleep(20 * time.Millisecond)
	err := check(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "underlying failure")
}

func TestContextCheck(t *testing.T) {
	called := false
	fn := func(ctx context.Context) error {
		called = true
		return nil
	}

	check := ContextCheck(fn)
	err := check(context.Background())

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestSimpleCheck(t *testing.T) {
	called := false
	fn := func() error {
		called = true
		return nil
	}

	check := SimpleCheck(fn)
	err := check(context.Background())

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestSimpleCheck_WithError(t *testing.T) {
	fn := func() error {
		return assert.AnError
	}

	check := SimpleCheck(fn)
	err := check(context.Background())

	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestCheck_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	check := ContextCheck(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	err := check(ctx)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestCheck_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	check := ContextCheck(func(ctx context.Context) error {
		select {
		case <-time.After(1 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	err := check(ctx)
	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}
