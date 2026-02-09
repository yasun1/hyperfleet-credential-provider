package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/errors"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

// ProviderFactory is a function that creates a new provider instance
type ProviderFactory func(ctx context.Context, logger logger.Logger) (Provider, error)

// Registry manages provider registration and instantiation
type Registry struct {
	mu        sync.RWMutex
	factories map[ProviderName]ProviderFactory
	logger    logger.Logger
}

// NewRegistry creates a new provider registry
func NewRegistry(logger logger.Logger) *Registry {
	return &Registry{
		factories: make(map[ProviderName]ProviderFactory),
		logger:    logger,
	}
}

// Register registers a provider factory
func (r *Registry) Register(name ProviderName, factory ProviderFactory) error {
	if !name.IsValid() {
		return errors.New(
			errors.ErrProviderNotSupported,
			fmt.Sprintf("invalid provider name: %s", name),
		).WithField("provider", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return errors.New(
			errors.ErrAlreadyExists,
			fmt.Sprintf("provider %s already registered", name),
		).WithField("provider", name)
	}

	r.factories[name] = factory
	r.logger.Info("Provider registered",
		logger.String("provider", name.String()),
	)

	return nil
}

// MustRegister registers a provider factory and panics on error
func (r *Registry) MustRegister(name ProviderName, factory ProviderFactory) {
	if err := r.Register(name, factory); err != nil {
		panic(err)
	}
}

func (r *Registry) Get(name ProviderName) (ProviderFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[name]
	if !exists {
		return nil, errors.New(
			errors.ErrProviderNotRegistered,
			fmt.Sprintf("provider %s not registered", name),
		).WithField("provider", name)
	}

	return factory, nil
}

func (r *Registry) Create(ctx context.Context, name ProviderName) (Provider, error) {
	factory, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	provider, err := factory(ctx, r.logger)
	if err != nil {
		return nil, errors.Wrap(
			errors.ErrProviderInitFailed,
			err,
			fmt.Sprintf("failed to create provider %s", name),
		).WithField("provider", name)
	}

	r.logger.Debug("Provider created",
		logger.String("provider", name.String()),
	)

	return provider, nil
}

// ListRegistered returns a list of registered provider names
func (r *Registry) ListRegistered() []ProviderName {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]ProviderName, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}

	return names
}

// IsRegistered checks if a provider is registered
func (r *Registry) IsRegistered(name ProviderName) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.factories[name]
	return exists
}

// Global registry instance
var globalRegistry *Registry
var globalRegistryOnce sync.Once

// GlobalRegistry returns the global provider registry
func GlobalRegistry(logger logger.Logger) *Registry {
	globalRegistryOnce.Do(func() {
		globalRegistry = NewRegistry(logger)
	})
	return globalRegistry
}

// Register registers a provider in the global registry
func Register(name ProviderName, factory ProviderFactory) error {
	return GlobalRegistry(logger.Nop()).Register(name, factory)
}

// MustRegister registers a provider in the global registry and panics on error
func MustRegister(name ProviderName, factory ProviderFactory) {
	GlobalRegistry(logger.Nop()).MustRegister(name, factory)
}
