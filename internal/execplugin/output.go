package execplugin

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/errors"
)

// OutputWriter handles writing ExecCredential output
type OutputWriter struct {
	writer io.Writer
}

// NewOutputWriter creates a new output writer
func NewOutputWriter(writer io.Writer) *OutputWriter {
	return &OutputWriter{
		writer: writer,
	}
}

// WriteToken writes a token as ExecCredential JSON to the output
func (w *OutputWriter) WriteToken(token *provider.Token) error {
	if token == nil {
		return errors.New(
			errors.ErrTokenInvalid,
			"token is nil",
		)
	}

	// Create ExecCredential response
	execCred := NewExecCredential(token.AccessToken, token.ExpiresAt)

	// Validate before writing
	if err := execCred.Validate(); err != nil {
		return errors.Wrap(
			errors.ErrExecPluginInvalidOutput,
			err,
			"failed to validate ExecCredential",
		)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(execCred, "", "  ")
	if err != nil {
		return errors.Wrap(
			errors.ErrExecPluginFailed,
			err,
			"failed to marshal ExecCredential to JSON",
		)
	}

	// Write to output
	if _, err := w.writer.Write(data); err != nil {
		return errors.Wrap(
			errors.ErrExecPluginFailed,
			err,
			"failed to write ExecCredential output",
		)
	}

	// Add newline for readability
	if _, err := w.writer.Write([]byte("\n")); err != nil {
		return errors.Wrap(
			errors.ErrExecPluginFailed,
			err,
			"failed to write newline",
		)
	}

	return nil
}

// WriteError writes an error to the output in a structured format
func (w *OutputWriter) WriteError(err error) error {
	// For exec plugins, errors should be written to stderr
	// This is a helper for debugging, not part of the exec plugin spec
	errMsg := fmt.Sprintf("Error: %v\n", err)
	if _, writeErr := w.writer.Write([]byte(errMsg)); writeErr != nil {
		return writeErr
	}
	return nil
}

// FormatToken formats a token as ExecCredential JSON string
func FormatToken(token *provider.Token) (string, error) {
	if token == nil {
		return "", errors.New(
			errors.ErrTokenInvalid,
			"token is nil",
		)
	}

	execCred := NewExecCredential(token.AccessToken, token.ExpiresAt)

	if err := execCred.Validate(); err != nil {
		return "", errors.Wrap(
			errors.ErrExecPluginInvalidOutput,
			err,
			"failed to validate ExecCredential",
		)
	}

	data, err := json.MarshalIndent(execCred, "", "  ")
	if err != nil {
		return "", errors.Wrap(
			errors.ErrExecPluginFailed,
			err,
			"failed to marshal ExecCredential to JSON",
		)
	}

	return string(data), nil
}
