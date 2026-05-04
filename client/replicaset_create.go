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

type CreateReplicaSetRequest struct {
	OrganizationID string
	ProjectID      string
	ClusterID      string
	ReplicaCount   int32 `json:"replicaCount"`
	Protected      bool  `json:"protected"`
}

func (c *Client) ReplicaSetCreate(
	ctx context.Context,
	req *CreateReplicaSetRequest,
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
		"replicaset",
	)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
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

	if resp.StatusCode != http.StatusCreated {
		return translateStatusCode(resp.StatusCode, "creating read-only replica set", resp.Body)
	}

	return nil
}
