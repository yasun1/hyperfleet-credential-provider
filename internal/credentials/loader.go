package credentials

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/errors"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/logger"
)

// Loader loads cloud provider credentials from various sources
type Loader interface {
	// LoadGCP loads GCP service account credentials
	LoadGCP(ctx context.Context, path string) (*GCPCredentials, error)

	// LoadAWS loads AWS credentials
	LoadAWS(ctx context.Context, opts AWSCredentialOptions) (*AWSCredentials, error)

	// LoadAzure loads Azure service principal credentials
	LoadAzure(ctx context.Context, opts AzureCredentialOptions) (*AzureCredentials, error)
}

// DefaultLoader implements Loader with standard credential loading
type DefaultLoader struct {
	logger logger.Logger
}

// NewLoader creates a new credential loader
func NewLoader(logger logger.Logger) Loader {
	return &DefaultLoader{
		logger: logger,
	}
}

// LoadGCP loads GCP service account credentials from a JSON file
func (l *DefaultLoader) LoadGCP(ctx context.Context, path string) (*GCPCredentials, error) {
	if path == "" {
		// Check GCP-standard environment variable
		path = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		if path == "" {
			return nil, errors.New(
				errors.ErrCredentialNotFound,
				"GCP credentials file path not provided",
			).WithDetail("set GOOGLE_APPLICATION_CREDENTIALS environment variable or use --credentials-file flag")
		}
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(
			errors.ErrCredentialLoadFailed,
			err,
			"failed to read GCP credentials file",
		).WithField("path", redactPath(path))
	}

	// Parse JSON
	var creds GCPCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, errors.Wrap(
			errors.ErrCredentialMalformed,
			err,
			"failed to parse GCP credentials JSON",
		).WithField("path", redactPath(path))
	}

	// Save raw JSON for SDK usage
	creds.RawJSON = string(data)

	// Validate required fields
	if err := l.validateGCPCredentials(&creds); err != nil {
		return nil, err
	}

	l.logger.Debug("GCP credentials loaded",
		logger.String("path", redactPath(path)),
		logger.String("project_id", creds.ProjectID),
		logger.String("client_email", creds.ClientEmail),
	)

	return &creds, nil
}

// LoadAWS loads AWS credentials from file or environment
func (l *DefaultLoader) LoadAWS(ctx context.Context, opts AWSCredentialOptions) (*AWSCredentials, error) {
	creds := &AWSCredentials{
		AccessKeyID:     opts.AccessKeyID,
		SecretAccessKey: opts.SecretAccessKey,
		SessionToken:    opts.SessionToken,
		Region:          opts.Region,
	}

	// Check for credentials file (priority: opts.CredentialsFile > AWS_CREDENTIALS_FILE)
	credentialsFile := opts.CredentialsFile
	if credentialsFile == "" {
		credentialsFile = os.Getenv("AWS_CREDENTIALS_FILE")
	}

	// If credentials file is specified, load from file
	if credentialsFile != "" {
		fileCreds, err := loadAWSFromFile(credentialsFile, opts.Profile)
		if err != nil {
			return nil, err
		}
		// File credentials take precedence
		creds.AccessKeyID = fileCreds.AccessKeyID
		creds.SecretAccessKey = fileCreds.SecretAccessKey
		creds.SessionToken = fileCreds.SessionToken
		if fileCreds.Region != "" {
			creds.Region = fileCreds.Region
		}
	} else if opts.UseEnvironment {
		// Load from individual environment variables
		if accessKey := os.Getenv("AWS_ACCESS_KEY_ID"); accessKey != "" {
			creds.AccessKeyID = accessKey
		}
		if secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY"); secretKey != "" {
			creds.SecretAccessKey = secretKey
		}
		if sessionToken := os.Getenv("AWS_SESSION_TOKEN"); sessionToken != "" {
			creds.SessionToken = sessionToken
		}
		if region := os.Getenv("AWS_REGION"); region != "" {
			creds.Region = region
		} else if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
			creds.Region = region
		}
	}

	// Validate
	if err := l.validateAWSCredentials(creds); err != nil {
		return nil, err
	}

	l.logger.Debug("AWS credentials loaded",
		logger.String("region", creds.Region),
		logger.Bool("has_session_token", creds.SessionToken != ""),
	)

	return creds, nil
}

// loadAWSFromFile loads AWS credentials from INI format file
func loadAWSFromFile(path string, profile string) (*AWSCredentials, error) {
	if profile == "" {
		profile = "default"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(
			errors.ErrCredentialLoadFailed,
			err,
			"failed to read AWS credentials file",
		).WithField("path", redactPath(path))
	}

	creds, err := parseAWSCredentialsINI(string(data), profile)
	if err != nil {
		return nil, errors.Wrap(
			errors.ErrCredentialMalformed,
			err,
			"failed to parse AWS credentials file",
		).WithField("path", redactPath(path))
	}

	return creds, nil
}

// parseAWSCredentialsINI parses AWS credentials in INI format
func parseAWSCredentialsINI(content string, profile string) (*AWSCredentials, error) {
	creds := &AWSCredentials{}
	inProfile := false
	profileHeader := "[" + profile + "]"

	lines := splitLines(content)
	for _, line := range lines {
		line = trimSpace(line)
		if line == "" || hasPrefix(line, "#") || hasPrefix(line, ";") {
			continue
		}

		if hasPrefix(line, "[") {
			inProfile = (line == profileHeader)
			continue
		}

		if !inProfile {
			continue
		}

		parts := splitKeyValue(line)
		if len(parts) != 2 {
			continue
		}

		key := trimSpace(parts[0])
		value := trimSpace(parts[1])

		switch key {
		case "aws_access_key_id":
			creds.AccessKeyID = value
		case "aws_secret_access_key":
			creds.SecretAccessKey = value
		case "aws_session_token":
			creds.SessionToken = value
		case "region":
			creds.Region = value
		}
	}

	return creds, nil
}

// LoadAzure loads Azure credentials from file or environment
func (l *DefaultLoader) LoadAzure(ctx context.Context, opts AzureCredentialOptions) (*AzureCredentials, error) {
	creds := &AzureCredentials{
		ClientID:     opts.ClientID,
		ClientSecret: opts.ClientSecret,
		TenantID:     opts.TenantID,
	}

	// Check for credentials file (priority: opts.CredentialsFile > AZURE_CREDENTIALS_FILE)
	credentialsFile := opts.CredentialsFile
	if credentialsFile == "" {
		credentialsFile = os.Getenv("AZURE_CREDENTIALS_FILE")
	}

	// If credentials file is specified, load from file
	if credentialsFile != "" {
		fileCreds, err := loadAzureFromFile(credentialsFile)
		if err != nil {
			return nil, err
		}
		// File credentials take precedence
		creds.ClientID = fileCreds.ClientID
		creds.ClientSecret = fileCreds.ClientSecret
		creds.TenantID = fileCreds.TenantID
	} else if opts.UseEnvironment {
		// Load from individual environment variables
		if clientID := os.Getenv("AZURE_CLIENT_ID"); clientID != "" {
			creds.ClientID = clientID
		}
		if clientSecret := os.Getenv("AZURE_CLIENT_SECRET"); clientSecret != "" {
			creds.ClientSecret = clientSecret
		}
		if tenantID := os.Getenv("AZURE_TENANT_ID"); tenantID != "" {
			creds.TenantID = tenantID
		}
	}

	// Validate
	if err := l.validateAzureCredentials(creds); err != nil {
		return nil, err
	}

	l.logger.Debug("Azure credentials loaded",
		logger.String("tenant_id", creds.TenantID),
		logger.String("client_id", creds.ClientID),
	)

	return creds, nil
}

// loadAzureFromFile loads Azure credentials from JSON file
func loadAzureFromFile(path string) (*AzureCredentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(
			errors.ErrCredentialLoadFailed,
			err,
			"failed to read Azure credentials file",
		).WithField("path", redactPath(path))
	}

	var creds struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		TenantID     string `json:"tenant_id"`
	}

	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, errors.Wrap(
			errors.ErrCredentialMalformed,
			err,
			"failed to parse Azure credentials JSON",
		).WithField("path", redactPath(path))
	}

	return &AzureCredentials{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		TenantID:     creds.TenantID,
	}, nil
}

// redactPath redacts sensitive parts of file paths for logging
func redactPath(path string) string {
	if path == "" {
		return ""
	}
	// Only show last component of path for security
	// Example: /vault/secrets/gcp-sa.json -> .../gcp-sa.json
	if len(path) > 20 {
		return "..." + path[len(path)-17:]
	}
	return path
}

// Helper functions for parsing INI files

func splitLines(content string) []string {
	return strings.Split(content, "\n")
}

func trimSpace(s string) string {
	return strings.TrimSpace(s)
}

func hasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

func splitKeyValue(line string) []string {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) == 2 {
		return parts
	}
	return []string{line}
}
