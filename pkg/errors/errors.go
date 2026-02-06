package errors

import (
	"encoding/json"
	"fmt"
)

// Error represents a structured application error following RFC 9457 (Problem Details)
type Error struct {
	// Type is a URI reference that identifies the error type
	Type string `json:"type"`

	// Title is a short, human-readable summary of the error
	Title string `json:"title"`

	// Status is the HTTP status code (used for categorization)
	Status int `json:"status"`

	// Detail is a human-readable explanation specific to this occurrence
	Detail string `json:"detail,omitempty"`

	// Instance is a URI reference that identifies the specific occurrence
	Instance string `json:"instance,omitempty"`

	// Code is an application-specific error code
	Code ErrorCode `json:"code"`

	// Cause is the underlying error that caused this error
	Cause error `json:"-"`

	// Fields contains additional context
	Fields map[string]interface{} `json:"fields,omitempty"`
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s", e.Title, e.Detail)
	}
	return e.Title
}

// Unwrap returns the underlying cause
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithDetail adds detail to the error
func (e *Error) WithDetail(detail string) *Error {
	e.Detail = detail
	return e
}

// WithCause adds a cause to the error
func (e *Error) WithCause(cause error) *Error {
	e.Cause = cause
	if e.Detail == "" && cause != nil {
		e.Detail = cause.Error()
	}
	return e
}

// WithField adds a field to the error context
func (e *Error) WithField(key string, value interface{}) *Error {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	e.Fields[key] = value
	return e
}

// WithFields adds multiple fields to the error context
func (e *Error) WithFields(fields map[string]interface{}) *Error {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	for k, v := range fields {
		e.Fields[k] = v
	}
	return e
}

// MarshalJSON implements json.Marshaler
func (e *Error) MarshalJSON() ([]byte, error) {
	type Alias Error
	return json.Marshal(&struct {
		*Alias
		CauseMsg string `json:"cause,omitempty"`
	}{
		Alias:    (*Alias)(e),
		CauseMsg: e.causeMessage(),
	})
}

// causeMessage returns the error message of the cause
func (e *Error) causeMessage() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return ""
}

// New creates a new Error
func New(code ErrorCode, title string) *Error {
	info := GetErrorInfo(code)
	return &Error{
		Type:   info.Type,
		Title:  title,
		Status: info.Status,
		Code:   code,
		Fields: make(map[string]interface{}),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(code ErrorCode, cause error, title string) *Error {
	return New(code, title).WithCause(cause)
}

// Is checks if the error is of a specific code
func Is(err error, code ErrorCode) bool {
	var appErr *Error
	if As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// As checks if the error is an application Error
func As(err error, target **Error) bool {
	for err != nil {
		if e, ok := err.(*Error); ok {
			*target = e
			return true
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return false
}

// GetCode extracts the error code from an error
func GetCode(err error) ErrorCode {
	var appErr *Error
	if As(err, &appErr) {
		return appErr.Code
	}
	return ErrUnknown
}

// GetStatus extracts the HTTP status from an error
func GetStatus(err error) int {
	var appErr *Error
	if As(err, &appErr) {
		return appErr.Status
	}
	return 500
}

// Redact creates a sanitized version of the error for external display
// This removes sensitive information that should not be exposed to users
func (e *Error) Redact() *Error {
	redacted := &Error{
		Type:   e.Type,
		Title:  e.Title,
		Status: e.Status,
		Code:   e.Code,
		Fields: make(map[string]interface{}),
	}

	// Only include safe fields
	for k, v := range e.Fields {
		if !isSensitiveField(k) {
			redacted.Fields[k] = v
		}
	}

	return redacted
}

// isSensitiveField checks if a field name indicates sensitive data
func isSensitiveField(field string) bool {
	sensitiveFields := []string{
		"password", "secret", "token", "key", "credential",
		"auth", "api_key", "private_key", "access_key",
	}

	for _, sensitive := range sensitiveFields {
		if field == sensitive {
			return true
		}
	}
	return false
}
