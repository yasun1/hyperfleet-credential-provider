package provider

import (
	"context"
	"time"
)

// MockProvider is a mock implementation of Provider for testing
type MockProvider struct {
	NameValue                string
	GetTokenFunc             func(ctx context.Context, opts GetTokenOptions) (*Token, error)
	ValidateCredentialsFunc  func(ctx context.Context) error
}

// GetToken implements Provider
func (m *MockProvider) GetToken(ctx context.Context, opts GetTokenOptions) (*Token, error) {
	if m.GetTokenFunc != nil {
		return m.GetTokenFunc(ctx, opts)
	}

	return &Token{
		AccessToken: "mock-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		TokenType:   "Bearer",
	}, nil
}

// ValidateCredentials implements Provider
func (m *MockProvider) ValidateCredentials(ctx context.Context) error {
	if m.ValidateCredentialsFunc != nil {
		return m.ValidateCredentialsFunc(ctx)
	}
	return nil
}

// Name implements Provider
func (m *MockProvider) Name() string {
	if m.NameValue != "" {
		return m.NameValue
	}
	return "mock"
}
