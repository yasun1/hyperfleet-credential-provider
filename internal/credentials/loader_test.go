package credentials

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadGCP(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Create temporary GCP credentials file
	tmpFile, err := os.CreateTemp("", "gcp-sa-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	gcpJSON := `{
		"type": "service_account",
		"project_id": "test-project",
		"private_key_id": "key123",
		"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
		"client_email": "test@test-project.iam.gserviceaccount.com",
		"client_id": "123456789",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test"
	}`
	_, err = tmpFile.WriteString(gcpJSON)
	require.NoError(t, err)
	tmpFile.Close()

	// Test loading
	creds, err := loader.LoadGCP(ctx, tmpFile.Name())
	require.NoError(t, err)
	assert.Equal(t, "test-project", creds.ProjectID)
	assert.Equal(t, "test@test-project.iam.gserviceaccount.com", creds.ClientEmail)
}

func TestLoadGCP_InvalidJSON(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Create temporary invalid JSON file
	tmpFile, err := os.CreateTemp("", "gcp-sa-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("{invalid json")
	require.NoError(t, err)
	tmpFile.Close()

	// Test loading should fail
	_, err = loader.LoadGCP(ctx, tmpFile.Name())
	assert.Error(t, err)
}

func TestLoadAWS_FromFile(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Create temporary AWS credentials file
	tmpFile, err := os.CreateTemp("", "aws-creds-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	awsINI := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
region = us-east-1

[test-profile]
aws_access_key_id = AKIATESTTESTTESTTEST
aws_secret_access_key = testSecretKeyForTestingPurposesOnly
aws_session_token = FwoGZXIvYXdzEBYaDH...TestSessionToken
region = us-west-2
`
	_, err = tmpFile.WriteString(awsINI)
	require.NoError(t, err)
	tmpFile.Close()

	t.Run("default profile", func(t *testing.T) {
		creds, err := loader.LoadAWS(ctx, AWSCredentialOptions{
			CredentialsFile: tmpFile.Name(),
			Profile:         "default",
		})
		require.NoError(t, err)
		assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", creds.AccessKeyID)
		assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", creds.SecretAccessKey)
		assert.Equal(t, "us-east-1", creds.Region)
		assert.Empty(t, creds.SessionToken)
	})

	t.Run("test-profile with session token", func(t *testing.T) {
		creds, err := loader.LoadAWS(ctx, AWSCredentialOptions{
			CredentialsFile: tmpFile.Name(),
			Profile:         "test-profile",
		})
		require.NoError(t, err)
		assert.Equal(t, "AKIATESTTESTTESTTEST", creds.AccessKeyID)
		assert.Equal(t, "testSecretKeyForTestingPurposesOnly", creds.SecretAccessKey)
		assert.Equal(t, "us-west-2", creds.Region)
		assert.Equal(t, "FwoGZXIvYXdzEBYaDH...TestSessionToken", creds.SessionToken)
	})

	t.Run("empty profile defaults to default", func(t *testing.T) {
		creds, err := loader.LoadAWS(ctx, AWSCredentialOptions{
			CredentialsFile: tmpFile.Name(),
		})
		require.NoError(t, err)
		assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", creds.AccessKeyID)
	})
}

func TestLoadAWS_FromEnvironment(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Set environment variables
	os.Setenv("AWS_ACCESS_KEY_ID", "AKIAENVENVENVENVENV")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "envSecretKeyForTestingPurposesOnly")
	os.Setenv("AWS_REGION", "eu-west-1")
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_REGION")
	}()

	creds, err := loader.LoadAWS(ctx, AWSCredentialOptions{
		UseEnvironment: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "AKIAENVENVENVENVENV", creds.AccessKeyID)
	assert.Equal(t, "envSecretKeyForTestingPurposesOnly", creds.SecretAccessKey)
	assert.Equal(t, "eu-west-1", creds.Region)
}

func TestLoadAWS_FileOverridesEnvironment(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Create temporary AWS credentials file
	tmpFile, err := os.CreateTemp("", "aws-creds-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	awsINI := `[default]
aws_access_key_id = AKIAFILFILFILFILFIL
aws_secret_access_key = fileSecretKeyForTestingPurposesOnly
region = ap-southeast-1
`
	_, err = tmpFile.WriteString(awsINI)
	require.NoError(t, err)
	tmpFile.Close()

	// Set environment variables
	os.Setenv("AWS_ACCESS_KEY_ID", "AKIAENVENVENVENVENV")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "envSecretKeyForTestingPurposesOnly")
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	}()

	// File should take precedence
	creds, err := loader.LoadAWS(ctx, AWSCredentialOptions{
		CredentialsFile: tmpFile.Name(),
		UseEnvironment:  true,
	})
	require.NoError(t, err)
	assert.Equal(t, "AKIAFILFILFILFILFIL", creds.AccessKeyID)
	assert.Equal(t, "fileSecretKeyForTestingPurposesOnly", creds.SecretAccessKey)
	assert.Equal(t, "ap-southeast-1", creds.Region)
}

func TestLoadAWS_NonExistentProfile(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Create temporary AWS credentials file
	tmpFile, err := os.CreateTemp("", "aws-creds-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	awsINI := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`
	_, err = tmpFile.WriteString(awsINI)
	require.NoError(t, err)
	tmpFile.Close()

	// Test loading non-existent profile
	_, err = loader.LoadAWS(ctx, AWSCredentialOptions{
		CredentialsFile: tmpFile.Name(),
		Profile:         "non-existent",
	})
	assert.Error(t, err)
}

func TestLoadAzure_FromFile(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Create temporary Azure credentials file
	tmpFile, err := os.CreateTemp("", "azure-creds-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	azureJSON := `{
		"client_id": "11111111-1111-1111-1111-111111111111",
		"client_secret": "test-client-secret-value",
		"tenant_id": "22222222-2222-2222-2222-222222222222"
	}`
	_, err = tmpFile.WriteString(azureJSON)
	require.NoError(t, err)
	tmpFile.Close()

	creds, err := loader.LoadAzure(ctx, AzureCredentialOptions{
		CredentialsFile: tmpFile.Name(),
	})
	require.NoError(t, err)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", creds.ClientID)
	assert.Equal(t, "test-client-secret-value", creds.ClientSecret)
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", creds.TenantID)
}

func TestLoadAzure_FromEnvironment(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Set environment variables
	os.Setenv("AZURE_CLIENT_ID", "33333333-3333-3333-3333-333333333333")
	os.Setenv("AZURE_CLIENT_SECRET", "env-client-secret-value")
	os.Setenv("AZURE_TENANT_ID", "44444444-4444-4444-4444-444444444444")
	defer func() {
		os.Unsetenv("AZURE_CLIENT_ID")
		os.Unsetenv("AZURE_CLIENT_SECRET")
		os.Unsetenv("AZURE_TENANT_ID")
	}()

	creds, err := loader.LoadAzure(ctx, AzureCredentialOptions{
		UseEnvironment: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "33333333-3333-3333-3333-333333333333", creds.ClientID)
	assert.Equal(t, "env-client-secret-value", creds.ClientSecret)
	assert.Equal(t, "44444444-4444-4444-4444-444444444444", creds.TenantID)
}

func TestLoadAzure_FileOverridesEnvironment(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Create temporary Azure credentials file
	tmpFile, err := os.CreateTemp("", "azure-creds-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	azureJSON := `{
		"client_id": "55555555-5555-5555-5555-555555555555",
		"client_secret": "file-client-secret-value",
		"tenant_id": "66666666-6666-6666-6666-666666666666"
	}`
	_, err = tmpFile.WriteString(azureJSON)
	require.NoError(t, err)
	tmpFile.Close()

	// Set environment variables
	os.Setenv("AZURE_CLIENT_ID", "33333333-3333-3333-3333-333333333333")
	os.Setenv("AZURE_CLIENT_SECRET", "env-client-secret-value")
	os.Setenv("AZURE_TENANT_ID", "44444444-4444-4444-4444-444444444444")
	defer func() {
		os.Unsetenv("AZURE_CLIENT_ID")
		os.Unsetenv("AZURE_CLIENT_SECRET")
		os.Unsetenv("AZURE_TENANT_ID")
	}()

	// File should take precedence
	creds, err := loader.LoadAzure(ctx, AzureCredentialOptions{
		CredentialsFile: tmpFile.Name(),
		UseEnvironment:  true,
	})
	require.NoError(t, err)
	assert.Equal(t, "55555555-5555-5555-5555-555555555555", creds.ClientID)
	assert.Equal(t, "file-client-secret-value", creds.ClientSecret)
	assert.Equal(t, "66666666-6666-6666-6666-666666666666", creds.TenantID)
}

func TestLoadAzure_InvalidJSON(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Create temporary invalid JSON file
	tmpFile, err := os.CreateTemp("", "azure-creds-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("{invalid json")
	require.NoError(t, err)
	tmpFile.Close()

	_, err = loader.LoadAzure(ctx, AzureCredentialOptions{
		CredentialsFile: tmpFile.Name(),
	})
	assert.Error(t, err)
}

func TestParseAWSCredentialsINI(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		profile     string
		expected    *AWSCredentials
		expectError bool
	}{
		{
			name: "basic default profile",
			content: `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`,
			profile: "default",
			expected: &AWSCredentials{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
		},
		{
			name: "profile with session token",
			content: `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
aws_session_token = FwoGZXIvYXdzEBYaDH...TestSessionToken
region = us-west-2
`,
			profile: "default",
			expected: &AWSCredentials{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				SessionToken:    "FwoGZXIvYXdzEBYaDH...TestSessionToken",
				Region:          "us-west-2",
			},
		},
		{
			name: "multiple profiles",
			content: `[default]
aws_access_key_id = DEFAULT_KEY
aws_secret_access_key = DEFAULT_SECRET

[prod]
aws_access_key_id = PROD_KEY
aws_secret_access_key = PROD_SECRET
`,
			profile: "prod",
			expected: &AWSCredentials{
				AccessKeyID:     "PROD_KEY",
				SecretAccessKey: "PROD_SECRET",
			},
		},
		{
			name: "with comments",
			content: `# This is a comment
[default]
; Another comment
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`,
			profile: "default",
			expected: &AWSCredentials{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
		},
		{
			name: "empty profile defaults to default",
			content: `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`,
			profile: "",
			expected: &AWSCredentials{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
		},
		{
			name: "non-existent profile",
			content: `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`,
			profile:     "non-existent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := parseAWSCredentialsINI(tt.content, tt.profile)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.AccessKeyID, creds.AccessKeyID)
				assert.Equal(t, tt.expected.SecretAccessKey, creds.SecretAccessKey)
				assert.Equal(t, tt.expected.SessionToken, creds.SessionToken)
				assert.Equal(t, tt.expected.Region, creds.Region)
			}
		})
	}
}

func TestRedactPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short path",
			input:    "/tmp/file.json",
			expected: "/tmp/file.json",
		},
		{
			name:     "long path",
			input:    "/vault/secrets/very/long/path/to/credentials.json",
			expected: ".../credentials.json",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadAWS_FromCredentialsFileEnv(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Create temporary AWS credentials file
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "aws-credentials")
	awsINI := `[default]
aws_access_key_id = AKIAENVFILENVFILENV
aws_secret_access_key = envFileSecretKeyForTestingPurposesOnly
region = ap-northeast-1
`
	err := os.WriteFile(credFile, []byte(awsINI), 0600)
	require.NoError(t, err)

	// Set environment variable
	os.Setenv("AWS_CREDENTIALS_FILE", credFile)
	defer os.Unsetenv("AWS_CREDENTIALS_FILE")

	// Load from environment variable
	creds, err := loader.LoadAWS(ctx, AWSCredentialOptions{})
	require.NoError(t, err)
	assert.Equal(t, "AKIAENVFILENVFILENV", creds.AccessKeyID)
	assert.Equal(t, "envFileSecretKeyForTestingPurposesOnly", creds.SecretAccessKey)
	assert.Equal(t, "ap-northeast-1", creds.Region)
}

func TestLoadAzure_FromCredentialsFileEnv(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Create temporary Azure credentials file
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "azure-credentials.json")
	azureJSON := `{
		"client_id": "77777777-7777-7777-7777-777777777777",
		"client_secret": "env-file-client-secret-value",
		"tenant_id": "88888888-8888-8888-8888-888888888888"
	}`
	err := os.WriteFile(credFile, []byte(azureJSON), 0600)
	require.NoError(t, err)

	// Set environment variable
	os.Setenv("AZURE_CREDENTIALS_FILE", credFile)
	defer os.Unsetenv("AZURE_CREDENTIALS_FILE")

	// Load from environment variable
	creds, err := loader.LoadAzure(ctx, AzureCredentialOptions{})
	require.NoError(t, err)
	assert.Equal(t, "77777777-7777-7777-7777-777777777777", creds.ClientID)
	assert.Equal(t, "env-file-client-secret-value", creds.ClientSecret)
	assert.Equal(t, "88888888-8888-8888-8888-888888888888", creds.TenantID)
}
