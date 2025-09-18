package client

import (
	"context"
	"encoding/json"
	"net/http"
	"path"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

type Acl struct {
	OrganizationID string         `json:"organizationId"`
	ProjectID      string         `json:"projectId"`
	CidrBlocks     []AclCidrBlock `json:"cidrBlocks"`
	Created        string         `json:"created"`
	Name           string         `json:"description"`
	Status         string         `json:"status"`
	Updated        string         `json:"updated"`
}

type GetAclRequest struct {
	OrganizationID string
	ProjectID      string
	AclID          string
}

type GetAclResponse struct {
	Acl Acl `json:"acl"`
}

func (c *Client) AclGet(
	ctx context.Context,
	req *GetAclRequest,
) (*GetAclResponse, diag.Diagnostics) {
	requestURL := *c.apiURL
	requestURL.Path = path.Join(
		"infra",
		"v1",
		"organizations",
		req.OrganizationID,
		"projects",
		req.ProjectID,
		"acls",
		req.AclID,
	)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, diag.Errorf("error constructing request: %w", err)
	}
	if err := c.addAuthorizationHeader(request); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, diag.Errorf("error sending request: %w", err)
	}
	defer closeIgnoreError(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, translateStatusCode(resp.StatusCode, "getting managed ACL", resp.Body)
	}

	decoder := json.NewDecoder(resp.Body)
	result := GetAclResponse{}
	if err := decoder.Decode(&result); err != nil {
		return nil, diag.Errorf("error parsing response: %w", err)
	}

	return &result, nil
}
