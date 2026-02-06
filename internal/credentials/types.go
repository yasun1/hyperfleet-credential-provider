package credentials

// GCPCredentials represents GCP service account credentials
type GCPCredentials struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`

	RawJSON string `json:"-"`
}

// AWSCredentials represents AWS credentials
type AWSCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string // Optional
	Region          string
}

// AzureCredentials represents Azure service principal credentials
type AzureCredentials struct {
	ClientID     string
	ClientSecret string
	TenantID     string
}

// AWSCredentialOptions holds options for loading AWS credentials
type AWSCredentialOptions struct {
	// CredentialsFile path to credentials file (takes precedence over environment)
	CredentialsFile string

	// AccessKeyID explicitly provided
	AccessKeyID string

	// SecretAccessKey explicitly provided
	SecretAccessKey string

	// SessionToken for temporary credentials
	SessionToken string

	// Region for AWS operations
	Region string

	// UseEnvironment determines if credentials should be loaded from environment
	UseEnvironment bool

	// UseSharedConfig determines if credentials should be loaded from ~/.aws/
	UseSharedConfig bool

	// Profile name for shared config
	Profile string
}

// AzureCredentialOptions holds options for loading Azure credentials
type AzureCredentialOptions struct {
	// CredentialsFile path to credentials file (takes precedence over environment)
	CredentialsFile string

	// ClientID explicitly provided
	ClientID string

	// ClientSecret explicitly provided
	ClientSecret string

	// TenantID explicitly provided
	TenantID string

	// UseEnvironment determines if credentials should be loaded from environment
	UseEnvironment bool

	// UseManagedIdentity determines if managed identity should be used
	UseManagedIdentity bool
}
