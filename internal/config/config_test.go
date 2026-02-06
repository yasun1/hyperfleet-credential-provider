package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "info", config.Log.Level)
	assert.Equal(t, "json", config.Log.Format)
	assert.Equal(t, 30*time.Second, config.Provider.Timeout)
	assert.True(t, config.Health.Enabled)
	assert.Equal(t, 8080, config.Health.Port)
	assert.True(t, config.Metrics.Enabled)
}

func TestConfigMerge(t *testing.T) {
	base := DefaultConfig()
	override := &Config{
		Log: LogConfig{
			Level: "debug",
		},
		Provider: ProviderConfig{
			Name:        "gcp",
			ClusterName: "test-cluster",
			GCP: &GCPConfig{
				ProjectID: "test-project",
			},
		},
	}

	base.Merge(override)

	assert.Equal(t, "debug", base.Log.Level)
	assert.Equal(t, "gcp", base.Provider.Name)
	assert.Equal(t, "test-cluster", base.Provider.ClusterName)
	assert.NotNil(t, base.Provider.GCP)
	assert.Equal(t, "test-project", base.Provider.GCP.ProjectID)
}

func TestFromFlags(t *testing.T) {
	config := FromFlags("aws", "my-cluster", "us-east-1", "debug", "console")

	assert.Equal(t, "aws", config.Provider.Name)
	assert.Equal(t, "my-cluster", config.Provider.ClusterName)
	assert.Equal(t, "us-east-1", config.Provider.Region)
	assert.Equal(t, "debug", config.Log.Level)
	assert.Equal(t, "console", config.Log.Format)
}
