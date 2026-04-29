package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

type ReadOnlyReplicaSet struct {
	OrganizationID string `json:"orgId"`
	ProjectID      string `json:"projectId"`
	ClusterID      string `json:"clusterId"`
	ReplicaCount   int32  `json:"replicaCount"`
	Status         string `json:"status"`
	Protected      bool   `json:"protected"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

type GetReplicaSetRequest struct {
	OrganizationID string
	ProjectID      string
	ClusterID      string
}

type GetReplicaSetResponse struct {
	ReadOnlyReplicaSet ReadOnlyReplicaSet `json:"readOnlyReplicaSet"`
}

func (c *Client) ReplicaSetGet(
	ctx context.Context,
	req *GetReplicaSetRequest,
) (*GetReplicaSetResponse, diag.Diagnostics) {
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
		"replicaset",
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

	if resp.StatusCode != http.StatusOK {
		return nil, translateStatusCode(resp.StatusCode, "getting read-only replica set", resp.Body)
	}

	decoder := json.NewDecoder(resp.Body)
	result := GetReplicaSetResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, diag.FromErr(fmt.Errorf("error parsing response: %w", err))
	}

	return &result, nil
}
