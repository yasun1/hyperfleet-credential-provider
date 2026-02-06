package aws

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"

	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/logger"
)

// ClusterInfo contains EKS cluster information
type ClusterInfo struct {
	// Endpoint is the cluster API server endpoint (with https://)
	Endpoint string

	// CertificateAuthority is the base64-encoded cluster CA certificate
	CertificateAuthority string

	// Version is the Kubernetes version
	Version string

	// Region is the AWS region
	Region string

	// ARN is the cluster ARN
	ARN string
}

// GetClusterInfo retrieves cluster information from EKS
func (p *Provider) GetClusterInfo(ctx context.Context, clusterName string) (*ClusterInfo, error) {
	p.logger.Info("Getting EKS cluster info",
		logger.String("cluster", clusterName),
		logger.String("region", p.config.Region),
	)

	// Load AWS credentials
	creds, err := p.credLoader.LoadAWS(ctx, p.awsCredOpts)
	if err != nil {
		p.logger.Error("Failed to load AWS credentials",
			logger.String("cluster", clusterName),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to load AWS credentials: %w", err)
	}

	// Create AWS config
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(creds.Region),
	)
	if err != nil {
		p.logger.Error("Failed to create AWS config",
			logger.String("cluster", clusterName),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	// Create EKS client
	eksClient := eks.NewFromConfig(cfg)

	p.logger.Debug("Fetching cluster details",
		logger.String("cluster", clusterName),
		logger.String("region", creds.Region),
	)

	// Describe cluster
	input := &eks.DescribeClusterInput{
		Name: &clusterName,
	}

	output, err := eksClient.DescribeCluster(ctx, input)
	if err != nil {
		p.logger.Error("Failed to describe cluster",
			logger.String("cluster", clusterName),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to describe cluster: %w", err)
	}

	cluster := output.Cluster
	if cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Validate cluster data
	if cluster.Endpoint == nil || *cluster.Endpoint == "" {
		return nil, fmt.Errorf("cluster endpoint is empty")
	}
	if cluster.CertificateAuthority == nil || cluster.CertificateAuthority.Data == nil {
		return nil, fmt.Errorf("cluster CA certificate is empty")
	}

	// Get CA certificate (already base64 encoded)
	caCert := *cluster.CertificateAuthority.Data

	info := &ClusterInfo{
		Endpoint:             *cluster.Endpoint,
		CertificateAuthority: caCert,
		Version:              getStringValue(cluster.Version),
		Region:               creds.Region,
		ARN:                  getStringValue(cluster.Arn),
	}

	p.logger.Info("Successfully retrieved cluster info",
		logger.String("cluster", clusterName),
		logger.String("endpoint", *cluster.Endpoint),
		logger.String("version", getStringValue(cluster.Version)),
		logger.String("region", creds.Region),
	)

	return info, nil
}

// getStringValue safely gets string value from pointer
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ValidateCertificate validates that the CA certificate is properly base64 encoded
func ValidateCertificate(cert string) error {
	_, err := base64.StdEncoding.DecodeString(cert)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %w", err)
	}
	return nil
}
