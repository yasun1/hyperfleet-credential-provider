# End-to-End (E2E) CLI Tests

## Overview

This directory contains end-to-end tests for the `hyperfleet-credential-provider` CLI commands.

## Test Coverage

### ✅ Tests Included

1. **TestVersionCommand** - Verify version command output
2. **TestHelpCommand** - Test help output for all commands
3. **TestGetTokenCommand_MissingFlags** - Validate required flag checks
4. **TestGetTokenCommand_WithEnvVars** - Verify environment variable support
5. **TestGenerateKubeconfigCommand_MissingFlags** - Validate required flag checks
6. **TestGenerateKubeconfigCommand_OutputFormat** - Test output format validation
7. **TestGetClusterInfoCommand_MissingFlags** - Validate required flag checks
8. **TestLogLevelFlag** - Test log level flag (debug, info, error)
9. **TestLogFormatFlag** - Test log format flag (json, console)
10. **TestPriorityFlagsOverEnv** - Verify flags override environment variables

## Running Tests

### Prerequisites

- Build the binary first: `make build`
- Binary must exist at `bin/hyperfleet-credential-provider`

### Run All E2E Tests

```bash
# Run all e2e tests
go test -v -tags=e2e ./test/e2e

# Run specific test
go test -v -tags=e2e ./test/e2e -run TestVersionCommand
```

### Build Tag

All tests in this directory use the `e2e` build tag:

```go
//go:build e2e
// +build e2e
```

This prevents them from running during normal `go test ./...` executions.

## Test Strategy

### What These Tests Validate

- ✅ Command-line argument parsing
- ✅ Environment variable support
- ✅ Flag priority (flags > env vars)
- ✅ Required flag validation
- ✅ Error message format
- ✅ Log level and format options
- ✅ Help output

### What These Tests Don't Validate

- ❌ Actual cloud provider API calls (see `test/integration/`)
- ❌ Token generation logic (see `test/integration/`)
- ❌ Kubeconfig content validation (would require real cluster access)

## Implementation Details

### runCommand Helper

All tests use a `runCommand` helper function that:
1. Locates the binary (or builds it if missing)
2. Executes the command with given args and env vars
3. Captures stdout and stderr
4. Returns output and error

```go
stdout, stderr, err := runCommand(t, args, env)
```

### Error Handling

- Commands with missing required flags return exit code 1
- Error messages are printed to stderr
- Tests check both stdout and stderr for error messages

## Examples

### Test Missing Required Flags

```go
func TestGetTokenCommand_MissingFlags(t *testing.T) {
    _, _, err := runCommand(t, []string{"get-token", "--cluster-name=test"}, nil)
    require.Error(t, err, "should fail with missing --provider")
}
```

### Test Environment Variables

```go
func TestGetTokenCommand_WithEnvVars(t *testing.T) {
    env := map[string]string{
        "HFCP_PROVIDER": "gcp",
        "HFCP_CLUSTER_NAME": "test-cluster",
    }

    _, _, err := runCommand(t, []string{"get-token"}, env)
    // Should not complain about missing flags
}
```

## Continuous Integration

These tests can be run in CI pipelines:

```yaml
# Example GitHub Actions workflow
- name: Run E2E Tests
  run: |
    make build
    go test -v -tags=e2e ./test/e2e
```

## Future Enhancements

Potential additions:
- Validate ExecCredential JSON output format
- Validate kubeconfig YAML structure
- Test with mock cloud credentials
- Performance benchmarks
