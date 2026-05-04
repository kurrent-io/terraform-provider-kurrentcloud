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

const replicaSetPollInterval = 5 * time.Second

func (c *Client) ReplicaSetWaitForState(
	ctx context.Context,
	req *WaitForReplicaSetStateRequest,
) diag.Diagnostics {
	start := time.Now()

	getRequest := &GetReplicaSetRequest{
		OrganizationID: req.OrganizationID,
		ProjectID:      req.ProjectID,
		ClusterID:      req.ClusterID,
	}

	ticker := time.NewTicker(replicaSetPollInterval)
	defer ticker.Stop()

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

		if status == StateDefunct {
			// Resources in a `defunct` state may not update their status
			// right away when being destroyed, so wait a bit before failing
			// the operation. Mirrors ManagedClusterWaitForState.
			if time.Since(start).Seconds() > 30.0 {
				return diag.Errorf(
					"read-only replica set entered a defunct state while waiting for %q",
					req.State,
				)
			}
		}

		select {
		case <-ctx.Done():
			return diag.FromErr(ctx.Err())
		case <-ticker.C:
		}
	}
}
