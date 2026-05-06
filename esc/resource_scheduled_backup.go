package esc

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/kurrent-io/terraform-provider-kurrentcloud/v2/client"
)

func resourceScheduledBackup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScheduledBackupCreate,
		ReadContext:   resourceScheduledBackupRead,
		DeleteContext: resourceScheduledBackupDelete,
		Description:   "Creates a new scheduled backup.",

		Importer: &schema.ResourceImporter{
			StateContext: resourceImport,
		},

		Schema: map[string]*schema.Schema{
			"description": {
				Description: "Human readable description of the job",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"project_id": {
				Description: "ID of the project in which the backup exists",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"schedule": {
				Description: "Schedule for the backup, defined using restricted subset of cron",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"source_cluster_id": {
				Description: "the ID of the cluster to back up",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"backup_description": {
				Description: "backup_description",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"max_backup_count": {
				Description: "The maximum number of backups to keep for this job",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeInt,
			},
		},
	}
}

func resourceEventstorecloudScheduledBackup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScheduledBackupCreate,
		ReadContext:   resourceScheduledBackupRead,
		DeleteContext: resourceScheduledBackupDelete,

		Description:        "Creates a new scheduled backup.",
		DeprecationMessage: "Use kurrentcloud_scheduled_backup instead. eventstorecloud_scheduled_backup will be removed in v3.0.0",

		Importer: &schema.ResourceImporter{
			StateContext: resourceImport,
		},

		Schema: map[string]*schema.Schema{
			"description": {
				Description: "Human readable description of the job",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"project_id": {
				Description: "ID of the project in which the backup exists",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"schedule": {
				Description: "Schedule for the backup, defined using restricted subset of cron",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"source_cluster_id": {
				Description: "the ID of the cluster to back up",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"backup_description": {
				Description: "backup_description",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"max_backup_count": {
				Description: "The maximum number of backups to keep for this job",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeInt,
			},
		},
	}
}

func resourceScheduledBackupCreate(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	projectId := d.Get("project_id").(string)

	request := client.CreateJobRequest{
		Data: map[string]interface{}{
			"clusterId":      d.Get("source_cluster_id").(string),
			"description":    d.Get("backup_description").(string),
			"maxBackupCount": d.Get("max_backup_count").(int),
		},
		Description: d.Get("description").(string),
		Schedule:    d.Get("schedule").(string),
		Type:        "ScheduledBackup",
	}

	resp, err := c.client.CreateJob(ctx, c.organizationId, projectId, request)
	if err != nil {
		return err
	}

	d.SetId(resp.Id)

	return resourceScheduledBackupRead(ctx, d, meta)
}

func resourceScheduledBackupRead(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)
	projectId := d.Get("project_id").(string)
	jobId := d.Id()

	var diags diag.Diagnostics

	resp, err := c.client.GetJob(ctx, c.organizationId, projectId, jobId)
	if err != nil {
		return diag.Errorf("Internal Server Error, try again later")
	}

	if resp.Job.Status == client.StateDeleted {
		d.SetId("")
		return nil
	}
	if err := d.Set("description", resp.Job.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("project_id", resp.Job.ProjectId); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("schedule", resp.Job.Schedule); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("source_cluster_id", resp.Job.Data["clusterId"]); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("backup_description", resp.Job.Data["description"]); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("max_backup_count", resp.Job.Data["maxBackupCount"]); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceScheduledBackupDelete(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	projectId := d.Get("project_id").(string)
	jobId := d.Id()

	if err := c.client.DeleteJob(ctx, c.organizationId, projectId, jobId); err != nil {
		return err
	}

	start := time.Now()
	for {
		resp, err := c.client.GetJob(ctx, c.organizationId, projectId, jobId)
		if err != nil {
			return diag.Errorf(
				"error polling job %q (%q) to see if it actually got deleted",
				jobId,
				d.Get("description"),
			)
		}
		elapsed := time.Since(start)
		if elapsed.Seconds() > 30.0 {
			return diag.Errorf(
				"job %q (%q) does not seem to be deleting",
				jobId,
				d.Get("description"),
			)
		}
		if resp.Job.Status == "deleted" {
			return nil
		}
		time.Sleep(1.0)
	}
}
