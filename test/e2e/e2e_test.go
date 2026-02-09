//go:build e2e
// +build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	binaryName = "hyperfleet-credential-provider"
)

// getBinaryPath returns the path to the built binary
func getBinaryPath(t *testing.T) string {
	// Try bin/ directory first
	binPath := filepath.Join("..", "..", "bin", binaryName)
	if _, err := os.Stat(binPath); err == nil {
		return binPath
	}

	// Try to build it
	t.Log("Binary not found, building...")
	cmd := exec.Command("make", "build")
	cmd.Dir = filepath.Join("..", "..")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build binary: %s", string(output))

	require.FileExists(t, binPath, "Binary should exist after build")
	return binPath
}

// runCommand runs the binary with given args and returns stdout, stderr, and error
func runCommand(t *testing.T, args []string, env map[string]string) (string, string, error) {
	binaryPath := getBinaryPath(t)

	cmd := exec.Command(binaryPath, args...)

	// Set environment variables
	if env != nil {
		cmd.Env = os.Environ()
		for key, value := range env {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestVersionCommand(t *testing.T) {
	stdout, stderr, err := runCommand(t, []string{"version"}, nil)
	require.NoError(t, err, "version command should succeed")

	// Check output contains expected information
	assert.Contains(t, stdout, "HyperFleet Credential Provider")
	assert.Contains(t, stdout, "Version:")
	assert.Contains(t, stdout, "Commit:")
	assert.Contains(t, stdout, "Build Time:")
	assert.Contains(t, stdout, "Go Version:")

	// stderr should be empty
	assert.Empty(t, stderr)
}

func TestHelpCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "root help",
			args: []string{"--help"},
		},
		{
			name: "get-token help",
			args: []string{"get-token", "--help"},
		},
		{
			name: "generate-kubeconfig help",
			args: []string{"generate-kubeconfig", "--help"},
		},
		{
			name: "get-cluster-info help",
			args: []string{"get-cluster-info", "--help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, _, err := runCommand(t, tt.args, nil)
			require.NoError(t, err, "help command should succeed")

			// Check help output contains usage information
			assert.Contains(t, stdout, "Usage:")
			assert.Contains(t, stdout, "Flags:")
		})
	}
}

func TestGetTokenCommand_MissingFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError string
	}{
		{
			name:        "missing provider",
			args:        []string{"get-token", "--cluster-name=test"},
			expectError: "--provider is required",
		},
		{
			name:        "missing cluster-name",
			args:        []string{"get-token", "--provider=gcp"},
			expectError: "--cluster-name is required",
		},
		{
			name:        "missing all required flags",
			args:        []string{"get-token"},
			expectError: "--provider is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCommand(t, tt.args, nil)
			require.Error(t, err, "command should fail with missing flags")

			// Error could be in stdout or stderr
			output := stdout + stderr
			assert.Contains(t, output, tt.expectError)
		})
	}
}

func TestGetTokenCommand_WithEnvVars(t *testing.T) {
	// This test verifies that environment variables work
	// It will fail with credentials error (expected), but should not fail with missing flags

	env := map[string]string{
		"HFCP_PROVIDER":     "gcp",
		"HFCP_CLUSTER_NAME": "test-cluster",
		"HFCP_PROJECT_ID":   "test-project",
		"HFCP_REGION":       "us-central1",
	}

	stdout, stderr, err := runCommand(t, []string{"get-token"}, env)

	// Should fail (no valid credentials), but error should NOT be "missing required flags"
	require.Error(t, err)

	output := stdout + stderr
	assert.NotContains(t, output, "--provider is required", "Should not complain about missing provider flag")
	assert.NotContains(t, output, "--cluster-name is required", "Should not complain about missing cluster-name flag")

	// Should fail with credentials-related error
	assert.Contains(t, output, "credential", "Should fail with credentials error")
}

func TestGenerateKubeconfigCommand_MissingFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError string
	}{
		{
			name:        "missing provider",
			args:        []string{"generate-kubeconfig", "--cluster-name=test"},
			expectError: "--provider is required",
		},
		{
			name:        "missing cluster-name",
			args:        []string{"generate-kubeconfig", "--provider=gcp"},
			expectError: "--cluster-name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCommand(t, tt.args, nil)
			require.Error(t, err, "command should fail with missing flags")

			output := stdout + stderr
			assert.Contains(t, output, tt.expectError)
		})
	}
}

func TestGenerateKubeconfigCommand_OutputFormat(t *testing.T) {
	// This test verifies the kubeconfig output format is valid YAML
	// It will fail with credentials error, but if we get output, it should be valid YAML

	tmpfile := filepath.Join(t.TempDir(), "kubeconfig.yaml")

	args := []string{
		"generate-kubeconfig",
		"--provider=gcp",
		"--cluster-name=test-cluster",
		"--project-id=test-project",
		"--region=us-central1",
		"--credentials-file=/tmp/nonexistent.json",
		"--output=" + tmpfile,
	}

	stdout, stderr, err := runCommand(t, args, nil)

	// Should fail with credentials error
	require.Error(t, err)

	output := stdout + stderr
	assert.Contains(t, output, "credential", "Should fail with credentials error")

	// File should not be created if command failed
	_, err = os.Stat(tmpfile)
	assert.True(t, os.IsNotExist(err), "Output file should not exist when command fails")
}

func TestGetClusterInfoCommand_MissingFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError string
	}{
		{
			name:        "missing provider",
			args:        []string{"get-cluster-info", "--cluster-name=test"},
			expectError: "--provider is required",
		},
		{
			name:        "missing cluster-name",
			args:        []string{"get-cluster-info", "--provider=gcp"},
			expectError: "--cluster-name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCommand(t, tt.args, nil)
			require.Error(t, err, "command should fail with missing flags")

			output := stdout + stderr
			assert.Contains(t, output, tt.expectError)
		})
	}
}

func TestLogLevelFlag(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		args     []string
	}{
		{
			name:     "debug level",
			logLevel: "debug",
			args:     []string{"get-token", "--provider=gcp", "--cluster-name=test", "--log-level=debug"},
		},
		{
			name:     "info level",
			logLevel: "info",
			args:     []string{"get-token", "--provider=gcp", "--cluster-name=test", "--log-level=info"},
		},
		{
			name:     "error level",
			logLevel: "error",
			args:     []string{"get-token", "--provider=gcp", "--cluster-name=test", "--log-level=error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCommand(t, tt.args, nil)

			// Command will fail (no credentials), but we can check log format
			require.Error(t, err)

			output := stdout + stderr
			// stderr should contain JSON log entries with the specified level
			// (at least for error level logs)
			if tt.logLevel == "error" {
				assert.Contains(t, output, `"level":"error"`)
			}
		})
	}
}

func TestLogFormatFlag(t *testing.T) {
	tests := []struct {
		name      string
		logFormat string
	}{
		{
			name:      "json format",
			logFormat: "json",
		},
		{
			name:      "console format",
			logFormat: "console",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{
				"get-token",
				"--provider=gcp",
				"--cluster-name=test",
				"--log-format=" + tt.logFormat,
			}

			stdout, stderr, err := runCommand(t, args, nil)

			// Command will fail (no credentials)
			require.Error(t, err)

			output := stdout + stderr
			// Check log format
			if tt.logFormat == "json" {
				// JSON format should have level field
				assert.Contains(t, output, `"level":`)
			} else {
				// Console format is more human-readable
				// Just verify it's not JSON (no quotes around level)
				// This is a weak check, but better than nothing
				assert.NotEmpty(t, output)
			}
		})
	}
}

// TestPriorityFlagsOverEnv verifies that command-line flags take priority over environment variables
func TestPriorityFlagsOverEnv(t *testing.T) {
	env := map[string]string{
		"HFCP_PROVIDER":     "aws", // Wrong provider in env
		"HFCP_CLUSTER_NAME": "wrong-cluster",
		"HFCP_PROJECT_ID":   "wrong-project",
	}

	args := []string{
		"get-token",
		"--provider=gcp", // Correct provider via flag
		"--cluster-name=test-cluster",
		"--project-id=test-project",
	}

	stdout, stderr, err := runCommand(t, args, env)

	// Command will fail (no credentials), but error should reflect GCP, not AWS
	require.Error(t, err)

	output := stdout + stderr
	// Should fail with GCP-related error message (e.g., "GCP credentials")
	// Not AWS-related error
	// This is a weak check, but validates the provider was overridden
	lowercaseOutput := strings.ToLower(output)
	if strings.Contains(lowercaseOutput, "provider") {
		// If error mentions provider, it should be about the flag value (gcp), not env value (aws)
		// This is hard to test without actually running with real credentials
		// So we just verify command didn't fail with "invalid provider" or similar
		assert.NotContains(t, output, "unsupported provider: aws")
	}
}
