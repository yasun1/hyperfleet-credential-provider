package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	err := New(ErrCredentialNotFound, "credential file not found")

	assert.NotNil(t, err)
	assert.Equal(t, ErrCredentialNotFound, err.Code)
	assert.Equal(t, "credential file not found", err.Title)
	assert.Equal(t, 404, err.Status)
	assert.Contains(t, err.Type, "credential-not-found")
}

func TestWrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := Wrap(ErrTokenGenerationFailed, cause, "failed to generate token")

	assert.NotNil(t, err)
	assert.Equal(t, ErrTokenGenerationFailed, err.Code)
	assert.Equal(t, "failed to generate token", err.Title)
	assert.Equal(t, cause, err.Cause)
	assert.Equal(t, cause.Error(), err.Detail)
}

func TestErrorWithDetail(t *testing.T) {
	err := New(ErrInvalidArgument, "invalid input").
		WithDetail("cluster name must not be empty")

	assert.Equal(t, "cluster name must not be empty", err.Detail)
	assert.Contains(t, err.Error(), "invalid input")
	assert.Contains(t, err.Error(), "cluster name must not be empty")
}

func TestErrorWithField(t *testing.T) {
	err := New(ErrClusterNotFound, "cluster not found").
		WithField("cluster_name", "my-cluster").
		WithField("region", "us-east-1")

	assert.Equal(t, "my-cluster", err.Fields["cluster_name"])
	assert.Equal(t, "us-east-1", err.Fields["region"])
}

func TestErrorWithFields(t *testing.T) {
	fields := map[string]interface{}{
		"provider": "gcp",
		"project":  "my-project",
	}

	err := New(ErrProviderInitFailed, "provider initialization failed").
		WithFields(fields)

	assert.Equal(t, "gcp", err.Fields["provider"])
	assert.Equal(t, "my-project", err.Fields["project"])
}

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := New(ErrInternal, "internal error").WithCause(cause)

	unwrapped := err.Unwrap()
	assert.Equal(t, cause, unwrapped)
	assert.True(t, errors.Is(err, cause))
}

func TestIs(t *testing.T) {
	err := New(ErrCredentialInvalid, "invalid credential")

	assert.True(t, Is(err, ErrCredentialInvalid))
	assert.False(t, Is(err, ErrTokenExpired))
}

func TestAs(t *testing.T) {
	err := New(ErrNetworkTimeout, "network timeout")

	var appErr *Error
	assert.True(t, As(err, &appErr))
	assert.Equal(t, ErrNetworkTimeout, appErr.Code)
}

func TestGetCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode ErrorCode
	}{
		{
			name:     "application error",
			err:      New(ErrRateLimitExceeded, "rate limit exceeded"),
			wantCode: ErrRateLimitExceeded,
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			wantCode: ErrUnknown,
		},
		{
			name:     "wrapped application error",
			err:      Wrap(ErrTokenInvalid, errors.New("cause"), "invalid token"),
			wantCode: ErrTokenInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := GetCode(tt.err)
			assert.Equal(t, tt.wantCode, code)
		})
	}
}

func TestGetStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "not found error",
			err:        New(ErrClusterNotFound, "cluster not found"),
			wantStatus: 404,
		},
		{
			name:       "unauthorized error",
			err:        New(ErrUnauthenticated, "unauthenticated"),
			wantStatus: 401,
		},
		{
			name:       "internal error",
			err:        New(ErrInternal, "internal error"),
			wantStatus: 500,
		},
		{
			name:       "standard error",
			err:        errors.New("standard error"),
			wantStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := GetStatus(tt.err)
			assert.Equal(t, tt.wantStatus, status)
		})
	}
}

func TestRedact(t *testing.T) {
	err := New(ErrCredentialInvalid, "invalid credential").
		WithField("username", "admin").
		WithField("password", "secret123").
		WithField("token", "abc123").
		WithField("cluster_name", "my-cluster")

	redacted := err.Redact()

	// Safe fields should be present
	assert.Equal(t, "my-cluster", redacted.Fields["cluster_name"])

	// Sensitive fields should be removed
	assert.NotContains(t, redacted.Fields, "password")
	assert.NotContains(t, redacted.Fields, "token")

	// Core error info should remain
	assert.Equal(t, err.Code, redacted.Code)
	assert.Equal(t, err.Title, redacted.Title)
	assert.Equal(t, err.Status, redacted.Status)
}

func TestGetErrorInfo(t *testing.T) {
	tests := []struct {
		name       string
		code       ErrorCode
		wantStatus int
		wantTitle  string
	}{
		{
			name:       "credential not found",
			code:       ErrCredentialNotFound,
			wantStatus: 404,
			wantTitle:  "Credential Not Found",
		},
		{
			name:       "token generation failed",
			code:       ErrTokenGenerationFailed,
			wantStatus: 500,
			wantTitle:  "Token Generation Failed",
		},
		{
			name:       "rate limit exceeded",
			code:       ErrRateLimitExceeded,
			wantStatus: 429,
			wantTitle:  "Rate Limit Exceeded",
		},
		{
			name:       "unknown code",
			code:       ErrorCode("INVALID"),
			wantStatus: 500,
			wantTitle:  "Unknown Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := GetErrorInfo(tt.code)
			assert.Equal(t, tt.wantStatus, info.Status)
			assert.Equal(t, tt.wantTitle, info.Title)
			assert.NotEmpty(t, info.Type)
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name         string
		code         ErrorCode
		wantRetryable bool
	}{
		{
			name:         "network timeout is retryable",
			code:         ErrNetworkTimeout,
			wantRetryable: true,
		},
		{
			name:         "network unreachable is retryable",
			code:         ErrNetworkUnreachable,
			wantRetryable: true,
		},
		{
			name:         "cluster unreachable is retryable",
			code:         ErrClusterUnreachable,
			wantRetryable: true,
		},
		{
			name:         "credential invalid is not retryable",
			code:         ErrCredentialInvalid,
			wantRetryable: false,
		},
		{
			name:         "permission denied is not retryable",
			code:         ErrPermissionDenied,
			wantRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryable := IsRetryable(tt.code)
			assert.Equal(t, tt.wantRetryable, retryable)
		})
	}
}

func TestErrorMarshalJSON(t *testing.T) {
	cause := errors.New("root cause")
	err := New(ErrTokenInvalid, "invalid token").
		WithDetail("token has expired").
		WithCause(cause).
		WithField("provider", "gcp")

	data, jsonErr := err.MarshalJSON()
	require.NoError(t, jsonErr)

	// Verify JSON contains expected fields
	jsonStr := string(data)
	assert.Contains(t, jsonStr, "ERR_TOKEN_INVALID")
	assert.Contains(t, jsonStr, "invalid token")
	assert.Contains(t, jsonStr, "token has expired")
	assert.Contains(t, jsonStr, "root cause")
	assert.Contains(t, jsonStr, "gcp")
}
