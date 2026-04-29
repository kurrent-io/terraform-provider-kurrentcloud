package client

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

type WaitForReplicaSetStateRequest struct {
	OrganizationID string
	ProjectID      string
	ClusterID      string
	State          string
}

// terminalReplicaSetFailureStates lists every replica-set status from the spec
// that means "this transition will not progress further" — when we see one
// while polling, the operation has failed and the waiter should stop.
var terminalReplicaSetFailureStates = map[string]struct{}{
	"compute create failed":       {},
	"delete failed":               {},
	"disk resize failed":          {},
	"upgrade failed":              {},
	"replica count change failed": {},
	"install failed":              {},
	"config change failed":        {},
}

func (c *Client) ReplicaSetWaitForState(
	ctx context.Context,
	req *WaitForReplicaSetStateRequest,
) diag.Diagnostics {
	getRequest := &GetReplicaSetRequest{
		OrganizationID: req.OrganizationID,
		ProjectID:      req.ProjectID,
		ClusterID:      req.ClusterID,
	}

	for {
		resp, err := c.ReplicaSetGet(ctx, getRequest)
		if err != nil {
			return err
		}

		status := resp.ReadOnlyReplicaSet.Status

		if status == req.State {
			return nil
		}

		if _, isFailure := terminalReplicaSetFailureStates[status]; isFailure {
			return diag.Errorf(
				"read-only replica set entered terminal failure state %q while waiting for %q",
				status,
				req.State,
			)
		}

		time.Sleep(5 * time.Second)
	}
}
