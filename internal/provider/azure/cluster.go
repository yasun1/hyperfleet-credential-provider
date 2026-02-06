package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v4"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

// ClusterInfo contains AKS cluster information
type ClusterInfo struct {
	// Endpoint is the cluster API server endpoint (with https://)
	Endpoint string

	// CertificateAuthority is the base64-encoded cluster CA certificate
	CertificateAuthority string

	// Version is the Kubernetes version
	Version string

	// Location is the Azure region
	Location string

	// ResourceID is the cluster resource ID
	ResourceID string
}

// GetClusterInfo retrieves cluster information from AKS
func (p *Provider) GetClusterInfo(ctx context.Context, clusterName, resourceGroup string) (*ClusterInfo, error) {
	p.logger.Info("Getting AKS cluster info",
		logger.String("cluster", clusterName),
		logger.String("resource_group", resourceGroup),
		logger.String("subscription", p.config.SubscriptionID),
	)

	// Load Azure credentials
	creds, err := p.credLoader.LoadAzure(ctx, p.azureCredOpts)
	if err != nil {
		p.logger.Error("Failed to load Azure credentials",
			logger.String("cluster", clusterName),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to load Azure credentials: %w", err)
	}

	// Create Azure credential
	credential, err := azidentity.NewClientSecretCredential(
		creds.TenantID,
		creds.ClientID,
		creds.ClientSecret,
		nil,
	)
	if err != nil {
		p.logger.Error("Failed to create Azure credential",
			logger.String("cluster", clusterName),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// Create AKS client
	clientFactory, err := armcontainerservice.NewClientFactory(p.config.SubscriptionID, credential, nil)
	if err != nil {
		p.logger.Error("Failed to create AKS client factory",
			logger.String("cluster", clusterName),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to create AKS client factory: %w", err)
	}

	managedClustersClient := clientFactory.NewManagedClustersClient()

	p.logger.Debug("Fetching cluster details",
		logger.String("cluster", clusterName),
		logger.String("resource_group", resourceGroup),
	)

	// Get managed cluster
	cluster, err := managedClustersClient.Get(ctx, resourceGroup, clusterName, nil)
	if err != nil {
		p.logger.Error("Failed to get cluster",
			logger.String("cluster", clusterName),
			logger.String("resource_group", resourceGroup),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	// Validate cluster data
	if cluster.Properties == nil {
		return nil, fmt.Errorf("cluster properties are nil")
	}
	if cluster.Properties.Fqdn == nil || *cluster.Properties.Fqdn == "" {
		return nil, fmt.Errorf("cluster FQDN is empty")
	}

	// Get admin credentials to extract CA certificate
	credResult, err := managedClustersClient.ListClusterAdminCredentials(ctx, resourceGroup, clusterName, nil)
	if err != nil {
		p.logger.Error("Failed to get cluster admin credentials",
			logger.String("cluster", clusterName),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to get cluster admin credentials: %w", err)
	}

	if len(credResult.Kubeconfigs) == 0 {
		return nil, fmt.Errorf("no kubeconfig found in admin credentials")
	}

	// Extract CA certificate from kubeconfig
	// The kubeconfig contains the base64-encoded CA cert
	caCert, err := extractCACertFromKubeconfig(credResult.Kubeconfigs[0].Value)
	if err != nil {
		p.logger.Error("Failed to extract CA certificate",
			logger.String("cluster", clusterName),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to extract CA certificate: %w", err)
	}

	// Build endpoint URL
	endpoint := "https://" + *cluster.Properties.Fqdn

	info := &ClusterInfo{
		Endpoint:             endpoint,
		CertificateAuthority: caCert,
		Version:              getStringValue(cluster.Properties.KubernetesVersion),
		Location:             getStringValue(cluster.Location),
		ResourceID:           getStringValue(cluster.ID),
	}

	p.logger.Info("Successfully retrieved cluster info",
		logger.String("cluster", clusterName),
		logger.String("endpoint", endpoint),
		logger.String("version", getStringValue(cluster.Properties.KubernetesVersion)),
		logger.String("location", getStringValue(cluster.Location)),
	)

	return info, nil
}

// extractCACertFromKubeconfig extracts the CA certificate from raw kubeconfig data
func extractCACertFromKubeconfig(kubeconfigData []byte) (string, error) {
	// Parse kubeconfig YAML to extract certificate-authority-data
	// For simplicity, we'll use string search since the format is predictable
	content := string(kubeconfigData)

	// Look for "certificate-authority-data: " in the kubeconfig
	const prefix = "certificate-authority-data: "
	start := -1
	for i := 0; i < len(content)-len(prefix); i++ {
		if content[i:i+len(prefix)] == prefix {
			start = i + len(prefix)
			break
		}
	}

	if start == -1 {
		return "", fmt.Errorf("certificate-authority-data not found in kubeconfig")
	}

	// Find the end of the line (certificate data)
	end := start
	for end < len(content) && content[end] != '\n' && content[end] != '\r' {
		end++
	}

	caCert := content[start:end]
	if len(caCert) == 0 {
		return "", fmt.Errorf("empty certificate-authority-data")
	}

	return caCert, nil
}

// getStringValue safely gets string value from pointer
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
