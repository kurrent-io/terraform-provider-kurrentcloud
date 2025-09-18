package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

type GetManagedClusterInitialCredentialsRequest struct {
	OrganizationID string
	ProjectID      string
	ClusterID      string
}

type GetManagedClusterInitialCredentialsResponse struct {
	AdminPassword string `json:"adminPassword"`
	OpsPassword   string `json:"opsPassword"`
	GeneratedAt   string `json:"generatedAt"`
	ClusterID     string `json:"clusterId"`
}

func (c *Client) ManagedClusterGetInitialCredentials(
	ctx context.Context,
	req *GetManagedClusterInitialCredentialsRequest,
) (*GetManagedClusterInitialCredentialsResponse, diag.Diagnostics) {
	requestURL := *c.apiURL
	requestURL.Path = path.Join(
		"mesdb",
		"v1",
		"organizations",
		req.OrganizationID,
		"projects",
		req.ProjectID,
		"clusters",
		req.ClusterID,
		"initialCredentials",
	)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, diag.FromErr(fmt.Errorf("error constructing request: %w", err))
	}
	if err := c.addAuthorizationHeader(request); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, diag.FromErr(fmt.Errorf("error sending request: %w", err))
	}
	defer closeIgnoreError(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return nil, diag.Errorf("initial credentials not found for cluster")
	}

	if resp.StatusCode == http.StatusPreconditionFailed {
		return nil, diag.Errorf("initial credentials have been cleared")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, translateStatusCode(
			resp.StatusCode,
			"getting cluster initial credentials",
			resp.Body,
		)
	}

	decoder := json.NewDecoder(resp.Body)
	result := GetManagedClusterInitialCredentialsResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, diag.FromErr(fmt.Errorf("error parsing response: %w", err))
	}

	return &result, nil
}
