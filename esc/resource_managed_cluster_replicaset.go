package esc

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/kurrent-io/terraform-provider-kurrentcloud/client"
)

func resourceManagedClusterReplicaset() *schema.Resource {
	return &schema.Resource{
		Description: "Manages the read-only replica set attached to a Kurrent Cloud managed cluster. The API exposes a single replica set per cluster, so the Terraform resource ID is the parent cluster ID.",

		CreateContext: resourceManagedClusterReplicasetCreate,
		ReadContext:   resourceManagedClusterReplicasetRead,
		UpdateContext: resourceManagedClusterReplicasetUpdate,
		DeleteContext: resourceManagedClusterReplicasetDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceManagedClusterReplicasetImport,
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Description: "ID of the project that owns the parent managed cluster",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"cluster_id": {
				Description: "ID of the parent managed cluster. Used as the resource ID since each cluster has at most one read-only replica set.",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"replica_count": {
				Description: "Number of read-only replicas in the set (1 or 2)",
				Required:    true,
				Type:        schema.TypeInt,
				ValidateDiagFunc: ValidateWithByPass(
					validation.ToDiagFunc(validation.IntBetween(1, 2)),
				),
			},
			"protected": {
				Description: "Protection from accidental deletion. Delete is rejected by the API while this is true.",
				Optional:    true,
				Default:     false,
				Type:        schema.TypeBool,
			},
			"status": {
				Description: "Current status of the read-only replica set",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"created_at": {
				Description: "Creation timestamp",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"updated_at": {
				Description: "Last modification timestamp",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceManagedClusterReplicasetCreate(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	projectId := d.Get("project_id").(string)
	clusterId := d.Get("cluster_id").(string)

	request := &client.CreateReplicaSetRequest{
		OrganizationID: c.organizationId,
		ProjectID:      projectId,
		ClusterID:      clusterId,
		ReplicaCount:   int32(d.Get("replica_count").(int)),
		Protected:      d.Get("protected").(bool),
	}

	if err := c.client.ReplicaSetCreate(ctx, request); err != nil {
		return err
	}

	d.SetId(clusterId)

	if err := c.client.ReplicaSetWaitForState(ctx, &client.WaitForReplicaSetStateRequest{
		OrganizationID: c.organizationId,
		ProjectID:      projectId,
		ClusterID:      clusterId,
		State:          "available",
	}); err != nil {
		return err
	}

	return resourceManagedClusterReplicasetRead(ctx, d, meta)
}

func resourceManagedClusterReplicasetRead(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	var diags diag.Diagnostics

	projectId := d.Get("project_id").(string)
	clusterId := d.Id()

	resp, err := c.client.ReplicaSetGet(ctx, &client.GetReplicaSetRequest{
		OrganizationID: c.organizationId,
		ProjectID:      projectId,
		ClusterID:      clusterId,
	})
	if err != nil {
		return err
	}

	if resp.ReadOnlyReplicaSet.Status == client.StateDeleted {
		d.SetId("")
		return nil
	}

	if err := d.Set("project_id", resp.ReadOnlyReplicaSet.ProjectID); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("cluster_id", resp.ReadOnlyReplicaSet.ClusterID); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("replica_count", int(resp.ReadOnlyReplicaSet.ReplicaCount)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("protected", resp.ReadOnlyReplicaSet.Protected); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("status", resp.ReadOnlyReplicaSet.Status); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("created_at", resp.ReadOnlyReplicaSet.CreatedAt); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("updated_at", resp.ReadOnlyReplicaSet.UpdatedAt); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceManagedClusterReplicasetUpdate(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	projectId := d.Get("project_id").(string)
	clusterId := d.Id()

	if d.HasChange("replica_count") || d.HasChange("protected") {
		request := &client.UpdateReplicaSetRequest{
			OrganizationID: c.organizationId,
			ProjectID:      projectId,
			ClusterID:      clusterId,
			ReplicaCount:   int32(d.Get("replica_count").(int)),
			Protected:      d.Get("protected").(bool),
		}

		if err := c.client.ReplicaSetUpdate(ctx, request); err != nil {
			return err
		}

		if err := c.client.ReplicaSetWaitForState(ctx, &client.WaitForReplicaSetStateRequest{
			OrganizationID: c.organizationId,
			ProjectID:      projectId,
			ClusterID:      clusterId,
			State:          "available",
		}); err != nil {
			return err
		}
	}

	return resourceManagedClusterReplicasetRead(ctx, d, meta)
}

func resourceManagedClusterReplicasetDelete(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	projectId := d.Get("project_id").(string)
	clusterId := d.Id()

	if err := c.client.ReplicaSetDelete(ctx, &client.DeleteReplicaSetRequest{
		OrganizationID: c.organizationId,
		ProjectID:      projectId,
		ClusterID:      clusterId,
	}); err != nil {
		return err
	}

	return c.client.ReplicaSetWaitForState(ctx, &client.WaitForReplicaSetStateRequest{
		OrganizationID: c.organizationId,
		ProjectID:      projectId,
		ClusterID:      clusterId,
		State:          client.StateDeleted,
	})
}

// resourceManagedClusterReplicasetImport accepts an import ID of
// "{project_id}:{cluster_id}". The cluster ID becomes the Terraform
// resource ID since each cluster has exactly one replica set.
func resourceManagedClusterReplicasetImport(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf(
			"failed to parse import id, expected format `{project_id}:{cluster_id}`",
		)
	}

	if err := d.Set("project_id", parts[0]); err != nil {
		return nil, err
	}
	if err := d.Set("cluster_id", parts[1]); err != nil {
		return nil, err
	}
	d.SetId(parts[1])

	return []*schema.ResourceData{d}, nil
}
