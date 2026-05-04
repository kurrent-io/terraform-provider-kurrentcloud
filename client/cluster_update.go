package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

type ExpandManagedClusterDiskRequest struct {
	OrganizationID string
	ProjectID      string
	ClusterID      string `json:"clusterId"`
	DiskIops       int32  `json:"diskIops,omitempty"`
	DiskSizeGB     int32  `json:"diskSizeGb"`
	DiskThroughput int32  `json:"diskThroughput,omitempty"`
	DiskType       string `json:"diskType"`
}

func (c *Client) ManagedClusterExpandDisk(
	ctx context.Context,
	req *ExpandManagedClusterDiskRequest,
) diag.Diagnostics {
	requestBody, err := json.Marshal(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error marshalling request: %w", err))
	}

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
		"disk",
		"expand",
	)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		requestURL.String(),
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error constructing request: %w", err))
	}
	request.Header.Add("Content-Type", "application/json")
	if err := c.addAuthorizationHeader(request); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error sending request: %w", err))
	}
	defer closeIgnoreError(resp.Body)

	if resp.StatusCode != http.StatusNoContent {
		return translateStatusCode(
			resp.StatusCode,
			"expanding disks for managed cluster",
			resp.Body,
		)
	}

	return nil
}

type ManagedClusterUpdateRequest struct {
	OrganizationID  string
	ProjectID       string
	ClusterID       string
	Description     string `json:"description"`
	Protected       bool   `json:"protected"`
	ProjectionLevel string `json:"projectionLevel,omitempty"`
}

func (c *Client) ManagedClusterUpdate(
	ctx context.Context,
	req *ManagedClusterUpdateRequest,
) diag.Diagnostics {
	requestBody, err := json.Marshal(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error marshalling request: %w", err))
	}

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
	)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		requestURL.String(),
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error constructing request: %w", err))
	}
	request.Header.Add("Content-Type", "application/json")
	if err := c.addAuthorizationHeader(request); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error sending request: %w", err))
	}
	defer closeIgnoreError(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return translateStatusCode(resp.StatusCode, "updating cluster", resp.Body)
	}

	return nil
}

type ManagedClusterResizeRequest struct {
	OrganizationID string
	ProjectID      string
	ClusterID      string
	TargetSize     string `json:"targetSize"`
}

func (c *Client) ManagedClusterResize(
	ctx context.Context,
	req *ManagedClusterResizeRequest,
) diag.Diagnostics {
	requestBody, err := json.Marshal(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error marshalling request: %v", err))
	}

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
		"commands",
		"resize",
	)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		requestURL.String(),
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error constructing request: %v", err))
	}
	request.Header.Add("Content-Type", "application/json")
	if err := c.addAuthorizationHeader(request); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error sending request: %v", err))
	}
	defer closeIgnoreError(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return translateStatusCode(resp.StatusCode, "resizing managed cluster", resp.Body)
	}

	return nil
}

type ManagedClusterUpgradeRequest struct {
	OrganizationID string
	ProjectID      string
	ClusterID      string
	TargetTag      string `json:"targetTag"`
}

func (c *Client) ManagedClusterUpgrade(
	ctx context.Context,
	req *ManagedClusterUpgradeRequest,
) diag.Diagnostics {
	requestBody, err := json.Marshal(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error marshalling request: %w", err))
	}

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
		"commands",
		"upgrade",
	)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		requestURL.String(),
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error constructing request: %w", err))
	}
	request.Header.Add("Content-Type", "application/json")
	if err := c.addAuthorizationHeader(request); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error sending request: %w", err))
	}
	defer closeIgnoreError(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return translateStatusCode(resp.StatusCode, "upgrading managed cluster", resp.Body)
	}

	return nil
}
