package gcp

import (
	"context"
	"fmt"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

// ClusterInfo contains GKE cluster information
type ClusterInfo struct {
	// Endpoint is the cluster API server endpoint (without https://)
	Endpoint string

	// CertificateAuthority is the base64-encoded cluster CA certificate
	CertificateAuthority string

	// Version is the Kubernetes version
	Version string

	// Location is the cluster location (region or zone)
	Location string
}

// GetClusterInfo retrieves cluster information from GKE
func (p *Provider) GetClusterInfo(ctx context.Context, clusterName, location string) (*ClusterInfo, error) {
	p.logger.Info("Getting GKE cluster info",
		logger.String("cluster", clusterName),
		logger.String("project", p.config.ProjectID),
		logger.String("location", location),
	)

	creds, err := p.credLoader.LoadGCP(ctx, p.config.CredentialsFile)
	if err != nil {
		p.logger.Error("Failed to load GCP credentials",
			logger.String("cluster", clusterName),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to load GCP credentials: %w", err)
	}

	gcpCreds, err := google.CredentialsFromJSON(ctx, []byte(creds.RawJSON), container.CloudPlatformScope)
	if err != nil {
		p.logger.Error("Failed to create GCP credentials",
			logger.String("cluster", clusterName),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to create GCP credentials: %w", err)
	}

	svc, err := container.NewService(ctx, option.WithCredentials(gcpCreds))
	if err != nil {
		p.logger.Error("Failed to create Container service",
			logger.String("cluster", clusterName),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to create Container service: %w", err)
	}

	// Build cluster resource name
	// Format: projects/{project}/locations/{location}/clusters/{cluster}
	name := fmt.Sprintf("projects/%s/locations/%s/clusters/%s",
		creds.ProjectID, location, clusterName)

	p.logger.Debug("Fetching cluster details",
		logger.String("resource_name", name),
	)

	// Get cluster details
	cluster, err := svc.Projects.Locations.Clusters.Get(name).Context(ctx).Do()
	if err != nil {
		p.logger.Error("Failed to get cluster info",
			logger.String("cluster", clusterName),
			logger.String("location", location),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to get cluster info: %w", err)
	}

	// Validate cluster data
	if cluster.Endpoint == "" {
		return nil, fmt.Errorf("cluster endpoint is empty")
	}
	if cluster.MasterAuth == nil || cluster.MasterAuth.ClusterCaCertificate == "" {
		return nil, fmt.Errorf("cluster CA certificate is empty")
	}

	info := &ClusterInfo{
		Endpoint:             cluster.Endpoint,
		CertificateAuthority: cluster.MasterAuth.ClusterCaCertificate,
		Version:              cluster.CurrentMasterVersion,
		Location:             cluster.Location,
	}

	p.logger.Info("Successfully retrieved cluster info",
		logger.String("cluster", clusterName),
		logger.String("endpoint", cluster.Endpoint),
		logger.String("version", cluster.CurrentMasterVersion),
		logger.String("location", cluster.Location),
	)

	return info, nil
}
