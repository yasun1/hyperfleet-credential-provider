package execplugin

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExecCredential is the response format for Kubernetes exec authentication plugins
// This follows the client.authentication.k8s.io/v1 API spec
type ExecCredential struct {
	// TypeMeta contains the API version and kind
	metav1.TypeMeta `json:",inline"`

	// Status contains the token and expiration
	Status *ExecCredentialStatus `json:"status,omitempty"`
}

// ExecCredentialStatus contains the token information
type ExecCredentialStatus struct {
	// ExpirationTimestamp is when the token expires (RFC3339)
	ExpirationTimestamp *metav1.Time `json:"expirationTimestamp,omitempty"`

	// Token is the bearer token for authentication
	Token string `json:"token"`

	// ClientCertificateData contains PEM-encoded client certificate (not used for tokens)
	ClientCertificateData string `json:"clientCertificateData,omitempty"`

	// ClientKeyData contains PEM-encoded client key (not used for tokens)
	ClientKeyData string `json:"clientKeyData,omitempty"`
}

// NewExecCredential creates a new ExecCredential response
func NewExecCredential(token string, expiresAt time.Time) *ExecCredential {
	return &ExecCredential{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "client.authentication.k8s.io/v1",
			Kind:       "ExecCredential",
		},
		Status: &ExecCredentialStatus{
			Token:               token,
			ExpirationTimestamp: &metav1.Time{Time: expiresAt},
		},
	}
}

// Validate validates the ExecCredential response
func (e *ExecCredential) Validate() error {
	if e.TypeMeta.APIVersion == "" {
		e.TypeMeta.APIVersion = "client.authentication.k8s.io/v1"
	}

	if e.TypeMeta.Kind == "" {
		e.TypeMeta.Kind = "ExecCredential"
	}

	if e.Status == nil {
		return &ValidationError{
			Field:   "status",
			Message: "status is required",
		}
	}

	if e.Status.Token == "" {
		return &ValidationError{
			Field:   "status.token",
			Message: "token is required",
		}
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
