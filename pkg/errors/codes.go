package errors

// ErrorCode represents an application-specific error code
type ErrorCode string

const (
	// Generic errors
	ErrUnknown          ErrorCode = "ERR_UNKNOWN"
	ErrInternal         ErrorCode = "ERR_INTERNAL"
	ErrInvalidArgument  ErrorCode = "ERR_INVALID_ARGUMENT"
	ErrNotFound         ErrorCode = "ERR_NOT_FOUND"
	ErrAlreadyExists    ErrorCode = "ERR_ALREADY_EXISTS"
	ErrPermissionDenied ErrorCode = "ERR_PERMISSION_DENIED"
	ErrUnauthenticated  ErrorCode = "ERR_UNAUTHENTICATED"

	// Credential errors
	ErrCredentialNotFound      ErrorCode = "ERR_CREDENTIAL_NOT_FOUND"
	ErrCredentialInvalid       ErrorCode = "ERR_CREDENTIAL_INVALID"
	ErrCredentialMalformed     ErrorCode = "ERR_CREDENTIAL_MALFORMED"
	ErrCredentialExpired       ErrorCode = "ERR_CREDENTIAL_EXPIRED"
	ErrCredentialLoadFailed    ErrorCode = "ERR_CREDENTIAL_LOAD_FAILED"
	ErrCredentialValidationFailed ErrorCode = "ERR_CREDENTIAL_VALIDATION_FAILED"

	// Token generation errors
	ErrTokenGenerationFailed ErrorCode = "ERR_TOKEN_GENERATION_FAILED"
	ErrTokenExpired          ErrorCode = "ERR_TOKEN_EXPIRED"
	ErrTokenInvalid          ErrorCode = "ERR_TOKEN_INVALID"
	ErrTokenMalformed        ErrorCode = "ERR_TOKEN_MALFORMED"

	// Provider errors
	ErrProviderNotSupported  ErrorCode = "ERR_PROVIDER_NOT_SUPPORTED"
	ErrProviderInitFailed    ErrorCode = "ERR_PROVIDER_INIT_FAILED"
	ErrProviderNotRegistered ErrorCode = "ERR_PROVIDER_NOT_REGISTERED"

	// Cluster errors
	ErrClusterNotFound      ErrorCode = "ERR_CLUSTER_NOT_FOUND"
	ErrClusterUnreachable   ErrorCode = "ERR_CLUSTER_UNREACHABLE"
	ErrClusterInvalidConfig ErrorCode = "ERR_CLUSTER_INVALID_CONFIG"

	// Configuration errors
	ErrConfigInvalid      ErrorCode = "ERR_CONFIG_INVALID"
	ErrConfigLoadFailed   ErrorCode = "ERR_CONFIG_LOAD_FAILED"
	ErrConfigMissingField ErrorCode = "ERR_CONFIG_MISSING_FIELD"

	// Network errors
	ErrNetworkTimeout     ErrorCode = "ERR_NETWORK_TIMEOUT"
	ErrNetworkUnreachable ErrorCode = "ERR_NETWORK_UNREACHABLE"
	ErrRateLimitExceeded  ErrorCode = "ERR_RATE_LIMIT_EXCEEDED"

	// Validation errors
	ErrValidationFailed ErrorCode = "ERR_VALIDATION_FAILED"
	ErrInvalidFormat    ErrorCode = "ERR_INVALID_FORMAT"
	ErrMissingRequired  ErrorCode = "ERR_MISSING_REQUIRED"

	// Exec plugin errors
	ErrExecPluginFailed       ErrorCode = "ERR_EXEC_PLUGIN_FAILED"
	ErrExecPluginInvalidOutput ErrorCode = "ERR_EXEC_PLUGIN_INVALID_OUTPUT"
)

// ErrorInfo contains metadata about an error code
type ErrorInfo struct {
	Code   ErrorCode
	Type   string
	Status int
	Title  string
}

// errorInfoMap maps error codes to their metadata
var errorInfoMap = map[ErrorCode]ErrorInfo{
	// Generic errors (500)
	ErrUnknown: {
		Code:   ErrUnknown,
		Type:   "https://hyperfleet.io/errors/unknown",
		Status: 500,
		Title:  "Unknown Error",
	},
	ErrInternal: {
		Code:   ErrInternal,
		Type:   "https://hyperfleet.io/errors/internal",
		Status: 500,
		Title:  "Internal Error",
	},

	// Client errors (400)
	ErrInvalidArgument: {
		Code:   ErrInvalidArgument,
		Type:   "https://hyperfleet.io/errors/invalid-argument",
		Status: 400,
		Title:  "Invalid Argument",
	},
	ErrValidationFailed: {
		Code:   ErrValidationFailed,
		Type:   "https://hyperfleet.io/errors/validation-failed",
		Status: 400,
		Title:  "Validation Failed",
	},
	ErrInvalidFormat: {
		Code:   ErrInvalidFormat,
		Type:   "https://hyperfleet.io/errors/invalid-format",
		Status: 400,
		Title:  "Invalid Format",
	},
	ErrMissingRequired: {
		Code:   ErrMissingRequired,
		Type:   "https://hyperfleet.io/errors/missing-required",
		Status: 400,
		Title:  "Missing Required Field",
	},

	// Not found errors (404)
	ErrNotFound: {
		Code:   ErrNotFound,
		Type:   "https://hyperfleet.io/errors/not-found",
		Status: 404,
		Title:  "Not Found",
	},
	ErrCredentialNotFound: {
		Code:   ErrCredentialNotFound,
		Type:   "https://hyperfleet.io/errors/credential-not-found",
		Status: 404,
		Title:  "Credential Not Found",
	},
	ErrClusterNotFound: {
		Code:   ErrClusterNotFound,
		Type:   "https://hyperfleet.io/errors/cluster-not-found",
		Status: 404,
		Title:  "Cluster Not Found",
	},
	ErrProviderNotRegistered: {
		Code:   ErrProviderNotRegistered,
		Type:   "https://hyperfleet.io/errors/provider-not-registered",
		Status: 404,
		Title:  "Provider Not Registered",
	},

	// Conflict errors (409)
	ErrAlreadyExists: {
		Code:   ErrAlreadyExists,
		Type:   "https://hyperfleet.io/errors/already-exists",
		Status: 409,
		Title:  "Already Exists",
	},

	// Authentication errors (401)
	ErrUnauthenticated: {
		Code:   ErrUnauthenticated,
		Type:   "https://hyperfleet.io/errors/unauthenticated",
		Status: 401,
		Title:  "Unauthenticated",
	},
	ErrCredentialInvalid: {
		Code:   ErrCredentialInvalid,
		Type:   "https://hyperfleet.io/errors/credential-invalid",
		Status: 401,
		Title:  "Invalid Credential",
	},
	ErrCredentialExpired: {
		Code:   ErrCredentialExpired,
		Type:   "https://hyperfleet.io/errors/credential-expired",
		Status: 401,
		Title:  "Credential Expired",
	},
	ErrTokenExpired: {
		Code:   ErrTokenExpired,
		Type:   "https://hyperfleet.io/errors/token-expired",
		Status: 401,
		Title:  "Token Expired",
	},
	ErrTokenInvalid: {
		Code:   ErrTokenInvalid,
		Type:   "https://hyperfleet.io/errors/token-invalid",
		Status: 401,
		Title:  "Invalid Token",
	},

	// Permission errors (403)
	ErrPermissionDenied: {
		Code:   ErrPermissionDenied,
		Type:   "https://hyperfleet.io/errors/permission-denied",
		Status: 403,
		Title:  "Permission Denied",
	},

	// Credential loading errors (500)
	ErrCredentialMalformed: {
		Code:   ErrCredentialMalformed,
		Type:   "https://hyperfleet.io/errors/credential-malformed",
		Status: 500,
		Title:  "Malformed Credential",
	},
	ErrCredentialLoadFailed: {
		Code:   ErrCredentialLoadFailed,
		Type:   "https://hyperfleet.io/errors/credential-load-failed",
		Status: 500,
		Title:  "Credential Load Failed",
	},
	ErrCredentialValidationFailed: {
		Code:   ErrCredentialValidationFailed,
		Type:   "https://hyperfleet.io/errors/credential-validation-failed",
		Status: 500,
		Title:  "Credential Validation Failed",
	},

	// Token generation errors (500)
	ErrTokenGenerationFailed: {
		Code:   ErrTokenGenerationFailed,
		Type:   "https://hyperfleet.io/errors/token-generation-failed",
		Status: 500,
		Title:  "Token Generation Failed",
	},
	ErrTokenMalformed: {
		Code:   ErrTokenMalformed,
		Type:   "https://hyperfleet.io/errors/token-malformed",
		Status: 500,
		Title:  "Malformed Token",
	},

	// Provider errors (500)
	ErrProviderNotSupported: {
		Code:   ErrProviderNotSupported,
		Type:   "https://hyperfleet.io/errors/provider-not-supported",
		Status: 400,
		Title:  "Provider Not Supported",
	},
	ErrProviderInitFailed: {
		Code:   ErrProviderInitFailed,
		Type:   "https://hyperfleet.io/errors/provider-init-failed",
		Status: 500,
		Title:  "Provider Initialization Failed",
	},

	// Cluster errors (500)
	ErrClusterUnreachable: {
		Code:   ErrClusterUnreachable,
		Type:   "https://hyperfleet.io/errors/cluster-unreachable",
		Status: 503,
		Title:  "Cluster Unreachable",
	},
	ErrClusterInvalidConfig: {
		Code:   ErrClusterInvalidConfig,
		Type:   "https://hyperfleet.io/errors/cluster-invalid-config",
		Status: 500,
		Title:  "Invalid Cluster Configuration",
	},

	// Configuration errors (500)
	ErrConfigInvalid: {
		Code:   ErrConfigInvalid,
		Type:   "https://hyperfleet.io/errors/config-invalid",
		Status: 500,
		Title:  "Invalid Configuration",
	},
	ErrConfigLoadFailed: {
		Code:   ErrConfigLoadFailed,
		Type:   "https://hyperfleet.io/errors/config-load-failed",
		Status: 500,
		Title:  "Configuration Load Failed",
	},
	ErrConfigMissingField: {
		Code:   ErrConfigMissingField,
		Type:   "https://hyperfleet.io/errors/config-missing-field",
		Status: 500,
		Title:  "Missing Configuration Field",
	},

	// Network errors (503/429)
	ErrNetworkTimeout: {
		Code:   ErrNetworkTimeout,
		Type:   "https://hyperfleet.io/errors/network-timeout",
		Status: 503,
		Title:  "Network Timeout",
	},
	ErrNetworkUnreachable: {
		Code:   ErrNetworkUnreachable,
		Type:   "https://hyperfleet.io/errors/network-unreachable",
		Status: 503,
		Title:  "Network Unreachable",
	},
	ErrRateLimitExceeded: {
		Code:   ErrRateLimitExceeded,
		Type:   "https://hyperfleet.io/errors/rate-limit-exceeded",
		Status: 429,
		Title:  "Rate Limit Exceeded",
	},

	// Exec plugin errors (500)
	ErrExecPluginFailed: {
		Code:   ErrExecPluginFailed,
		Type:   "https://hyperfleet.io/errors/exec-plugin-failed",
		Status: 500,
		Title:  "Exec Plugin Failed",
	},
	ErrExecPluginInvalidOutput: {
		Code:   ErrExecPluginInvalidOutput,
		Type:   "https://hyperfleet.io/errors/exec-plugin-invalid-output",
		Status: 500,
		Title:  "Invalid Exec Plugin Output",
	},
}

// GetErrorInfo returns metadata for an error code
func GetErrorInfo(code ErrorCode) ErrorInfo {
	if info, ok := errorInfoMap[code]; ok {
		return info
	}
	return errorInfoMap[ErrUnknown]
}

// IsRetryable returns true if the error code indicates a retryable error
func IsRetryable(code ErrorCode) bool {
	retryableCodes := []ErrorCode{
		ErrNetworkTimeout,
		ErrNetworkUnreachable,
		ErrClusterUnreachable,
	}

	for _, retryable := range retryableCodes {
		if code == retryable {
			return true
		}
	}
	return false
}
