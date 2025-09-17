package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

type AclCidrBlock struct {
	Address string `json:"address"`
	Comment string `json:"comment"`
}

type CreateAclRequest struct {
	OrganizationID string
	ProjectID      string
	Name           string         `json:"description"`
	CidrBlocks     []AclCidrBlock `json:"cidrBlocks"`
}

type CreateAclResponse struct {
	AclID string `json:"id"`
}

func (c *Client) AclCreate(
	ctx context.Context,
	req *CreateAclRequest,
) (*CreateAclResponse, diag.Diagnostics) {
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, diag.Errorf("error marshalling request: %w", err)
	}

	requestURL := *c.apiURL
	requestURL.Path = path.Join(
		"infra",
		"v1",
		"organizations",
		req.OrganizationID,
		"projects",
		req.ProjectID,
		"acls",
	)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		requestURL.String(),
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return nil, diag.Errorf("error constructing request: %w", err)
	}
	request.Header.Add("Content-Type", "application/json")
	if err := c.addAuthorizationHeader(request); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, diag.Errorf("error sending request: %w", err)
	}
	defer closeIgnoreError(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, translateStatusCode(resp.StatusCode, "creating acl", resp.Body)
	}

	decoder := json.NewDecoder(resp.Body)
	result := CreateAclResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, diag.Errorf("error parsing response: %w", err)
	}

	return &result, nil
}
