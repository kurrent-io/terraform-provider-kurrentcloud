package esc

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/kurrent-io/terraform-provider-kurrentcloud/client"
)

func resourceIntegrationAwsCloudWatchLogs() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIntegrationAwsCloudWatchLogsCreate,
		ReadContext:   resourceIntegrationAwsCloudWatchLogsRead,
		DeleteContext: resourceIntegrationAwsCloudWatchLogsDelete,
		UpdateContext: resourceIntegrationAwsCloudWatchLogsUpdate,

		Description: `Manages integrations for AwsCloudWatch logs.

**NOTE**: This functionality is currently in beta. To access it please contact support.`,
		Importer: &schema.ResourceImporter{
			StateContext: resourceImport,
		},

		Schema: map[string]*schema.Schema{
			"access_key_id": {
				Description: "The access key ID of IAM credentials which have permissions to create and write to the log group",
				Required:    false,
				ForceNew:    false,
				Optional:    true,
				Sensitive:   true,
				Type:        schema.TypeString,
			},
			"cluster_ids": {
				Description: "Clusters to be used with this integration",
				Required:    true,
				ForceNew:    false,
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"description": {
				Description: "Human readable description of the integration",
				Required:    true,
				ForceNew:    false,
				Type:        schema.TypeString,
			},
			"group_name": {
				Description: "Name of the CloudWatch group",
				Required:    true,
				ForceNew:    false,
				Sensitive:   false,
				Type:        schema.TypeString,
			},
			"project_id": {
				Description: "ID of the project to which the integration applies",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"region": {
				Description: "AWS region for group",
				Required:    true,
				ForceNew:    false,
				Sensitive:   false,
				Type:        schema.TypeString,
			},
			"secret_access_key": {
				Description: "The secret access key of IAM credentials which will be used to write to the log groups",
				Required:    false,
				ForceNew:    false,
				Optional:    true,
				Sensitive:   true,
				Type:        schema.TypeString,
			},
		},
	}
}

func resourceEventstorecloudIntegrationAwsCloudWatchLogs() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIntegrationAwsCloudWatchLogsCreate,
		ReadContext:   resourceIntegrationAwsCloudWatchLogsRead,
		DeleteContext: resourceIntegrationAwsCloudWatchLogsDelete,
		UpdateContext: resourceIntegrationAwsCloudWatchLogsUpdate,

		Description: `Manages integrations for AwsCloudWatch logs.

**NOTE**: This functionality is currently in beta. To access it please contact support.`,
		DeprecationMessage: "Use kurrentcloud_integration_awscloudwatch_logs instead. eventstorecloud_integration_awscloudwatch_logs will be removed in v3.0.0",
		Importer: &schema.ResourceImporter{
			StateContext: resourceImport,
		},

		Schema: map[string]*schema.Schema{
			"access_key_id": {
				Description: "The access key ID of IAM credentials which have permissions to create and write to the log group",
				Required:    false,
				ForceNew:    false,
				Optional:    true,
				Sensitive:   true,
				Type:        schema.TypeString,
			},
			"cluster_ids": {
				Description: "Clusters to be used with this integration",
				Required:    true,
				ForceNew:    false,
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"description": {
				Description: "Human readable description of the integration",
				Required:    true,
				ForceNew:    false,
				Type:        schema.TypeString,
			},
			"group_name": {
				Description: "Name of the CloudWatch group",
				Required:    true,
				ForceNew:    false,
				Sensitive:   false,
				Type:        schema.TypeString,
			},
			"project_id": {
				Description: "ID of the project to which the integration applies",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"region": {
				Description: "AWS region for group",
				Required:    true,
				ForceNew:    false,
				Sensitive:   false,
				Type:        schema.TypeString,
			},
			"secret_access_key": {
				Description: "The secret access key of IAM credentials which will be used to write to the log groups",
				Required:    false,
				ForceNew:    false,
				Optional:    true,
				Sensitive:   true,
				Type:        schema.TypeString,
			},
		},
	}
}

func resourceIntegrationAwsCloudWatchLogsCreate(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	projectId := d.Get("project_id").(string)
	accessKeyId := d.Get("access_key_id").(string)
	secretAccessKey := d.Get("secret_access_key").(string)
	if accessKeyId == "" || secretAccessKey == "" {
		var diags diag.Diagnostics
		if accessKeyId == "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Missing access_key_id.",
			})
		}
		if secretAccessKey == "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Missing secret_access_key.",
			})
		}
		return diags
	}

	data := map[string]interface{}{
		"accessKeyId":     accessKeyId,
		"clusterIds":      interfaceToStringList(d.Get("cluster_ids")),
		"groupName":       d.Get("group_name").(string),
		"region":          d.Get("region").(string),
		"secretAccessKey": secretAccessKey,
		"source":          "logs",
		"sink":            "awsCloudWatchLogs",
	}
	request := client.CreateIntegrationRequest{
		Data:        data,
		Description: d.Get("description").(string),
	}

	resp, err := c.client.CreateIntegration(ctx, c.organizationId, projectId, request)
	if err != nil {
		return err
	}

	d.SetId(resp.Id)

	return resourceIntegrationAwsCloudWatchLogsRead(ctx, d, meta)
}

func resourceIntegrationAwsCloudWatchLogsRead(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	var diags diag.Diagnostics

	projectId := d.Get("project_id").(string)
	integrationId := d.Id()

	resp, err := c.client.GetIntegration(ctx, c.organizationId, projectId, integrationId)
	if err != nil {
		return diag.Errorf("Internal Server Error, try again later")
	}

	if resp.Integration.Status == client.StateDeleted {
		d.SetId("")
		return nil
	}

	if err := d.Set("description", resp.Integration.Description); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("project_id", resp.Integration.ProjectId); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	setVal := func(tfName, dataKey string) {
		if val, ok := resp.Integration.Data[dataKey]; ok {
			if err := d.Set(tfName, val); err != nil {
				diags = append(diags, diag.FromErr(err)...)
			}
		} else {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("returned API data value missing key %q", dataKey),
			})
		}
	}
	setVal("cluster_ids", "clusterIds")
	setVal("group_name", "groupName")
	setVal("region", "region")

	return diags
}

func resourceIntegrationAwsCloudWatchLogsDelete(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	projectId := d.Get("project_id").(string)
	integrationId := d.Id()

	if err := c.client.DeleteIntegration(ctx, c.organizationId, projectId, integrationId); err != nil {
		return err
	}

	start := time.Now()
	for {
		resp, err := c.client.GetIntegration(ctx, c.organizationId, projectId, integrationId)
		if err != nil {
			return diag.Errorf(
				"error polling integration %q (%q) to see if it actually got deleted",
				integrationId,
				d.Get("description"),
			)
		}
		elapsed := time.Since(start)
		if elapsed.Seconds() > 30.0 {
			return diag.Errorf(
				"integration %q (%q) does not seem to be deleting",
				integrationId,
				d.Get("description"),
			)
		}
		if resp.Integration.Status == "deleted" {
			return nil
		}
		time.Sleep(1.0)
	}
}

func resourceIntegrationAwsCloudWatchLogsUpdate(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	if !d.HasChanges(
		"cluster_ids",
		"description",
		"group_name",
		"region",
		"access_key_id",
		"secret_access_key",
	) {
		return resourceIntegrationAwsCloudWatchLogsRead(ctx, d, meta)
	}

	var desc *string
	desc = nil
	newDesc := d.Get("description").(string)
	if newDesc != "" {
		desc = &newDesc
	}

	data := map[string]interface{}{
		"clusterIds": interfaceToStringList(d.Get("cluster_ids")),
		"groupName":  d.Get("group_name").(string),
		"source":     "logs",
		"sink":       "awsCloudWatchLogs",
		"region":     d.Get("region").(string),
	}
	if d.HasChange("access_key_id") {
		newVal := d.Get("access_key_id").(string)
		if newVal != "" {
			data["accessKeyId"] = newVal
		}
	}
	if d.HasChange("secret_access_key") {
		newVal := d.Get("secret_access_key").(string)
		if newVal != "" {
			data["secretAccessKey"] = newVal
		}
	}

	request := client.UpdateIntegrationRequest{
		Data:        &data,
		Description: desc,
	}

	orgId := c.organizationId
	projectId := d.Get("project_id").(string)
	integrationId := d.Id()

	if err := c.client.UpdateIntegration(ctx, orgId, projectId, integrationId, request); err != nil {
		return err
	}

	return resourceIntegrationAwsCloudWatchLogsRead(ctx, d, meta)
}
