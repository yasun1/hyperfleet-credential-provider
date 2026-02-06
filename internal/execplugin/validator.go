package execplugin

import (
	"time"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/errors"
)

// Validator validates ExecCredential responses
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateExecCredential validates an ExecCredential response
func (v *Validator) ValidateExecCredential(cred *ExecCredential) error {
	if cred == nil {
		return errors.New(
			errors.ErrExecPluginInvalidOutput,
			"ExecCredential is nil",
		)
	}

	// Validate API version
	if cred.TypeMeta.APIVersion != "client.authentication.k8s.io/v1" &&
		cred.TypeMeta.APIVersion != "client.authentication.k8s.io/v1beta1" {
		return errors.New(
			errors.ErrExecPluginInvalidOutput,
			"invalid API version",
		).WithField("apiVersion", cred.TypeMeta.APIVersion).
			WithDetail("expected client.authentication.k8s.io/v1")
	}

	// Validate kind
	if cred.TypeMeta.Kind != "ExecCredential" {
		return errors.New(
			errors.ErrExecPluginInvalidOutput,
			"invalid kind",
		).WithField("kind", cred.TypeMeta.Kind).
			WithDetail("expected ExecCredential")
	}

	// Validate status
	if cred.Status == nil {
		return errors.New(
			errors.ErrExecPluginInvalidOutput,
			"status is required",
		)
	}

	// Validate token
	if cred.Status.Token == "" {
		return errors.New(
			errors.ErrExecPluginInvalidOutput,
			"token is required",
		)
	}

	// Validate expiration timestamp (if present)
	if cred.Status.ExpirationTimestamp != nil {
		expiresAt := cred.Status.ExpirationTimestamp.Time
		if expiresAt.Before(time.Now()) {
			return errors.New(
				errors.ErrTokenExpired,
				"token has already expired",
			).WithField("expires_at", expiresAt)
		}
	}

	return nil
}
