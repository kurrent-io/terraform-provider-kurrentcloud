package esc

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/kurrent-io/terraform-provider-kurrentcloud/client"
)

func resourceIntegration() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIntegrationCreate,
		ReadContext:   resourceIntegrationRead,
		DeleteContext: resourceIntegrationDelete,
		UpdateContext: resourceIntegrationUpdate,

		Description: "Manages integration resources, for example Slack or OpsGenie.",

		Importer: &schema.ResourceImporter{
			StateContext: resourceImport,
		},

		Schema: map[string]*schema.Schema{
			"description": {
				Description: "Human readable description of the integration",
				Required:    true,
				ForceNew:    false,
				Type:        schema.TypeString,
			},
			"project_id": {
				Description: "ID of the project to which the integration applies",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"data": {
				Description: "Data for the integration",
				Required:    true,
				ForceNew:    false,
				Type:        schema.TypeMap,
			},
		},
	}
}

func resourceEventstorecloudIntegration() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIntegrationCreate,
		ReadContext:   resourceIntegrationRead,
		DeleteContext: resourceIntegrationDelete,
		UpdateContext: resourceIntegrationUpdate,

		Description:        "Manages integration resources, for example Slack or OpsGenie.",
		DeprecationMessage: "Use kurrentcloud_integration instead. eventstorecloud_integration will be removed in v3.0.0",

		Importer: &schema.ResourceImporter{
			StateContext: resourceImport,
		},

		Schema: map[string]*schema.Schema{
			"description": {
				Description: "Human readable description of the integration",
				Required:    true,
				ForceNew:    false,
				Type:        schema.TypeString,
			},
			"project_id": {
				Description: "ID of the project to which the integration applies",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"data": {
				Description: "Data for the integration",
				Required:    true,
				ForceNew:    false,
				Type:        schema.TypeMap,
			},
		},
	}
}

type ModifyMapArgs struct {
	RenameNames []struct{ from, to string }
	RemoveNames []string
}

func modifyMap(args ModifyMapArgs, data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range data {
		result[k] = v
	}

	for _, e := range args.RenameNames {
		if value, ok := result[e.from]; ok {
			result[e.to] = value
			delete(result, e.from)
		}
	}
	for _, name := range args.RemoveNames {
		delete(result, name)
	}
	return result
}

func translateTfDataToApi(data map[string]interface{}) map[string]interface{} {
	return modifyMap(ModifyMapArgs{
		RenameNames: []struct{ from, to string }{
			{from: "access_key_id", to: "accessKeyId"},
			{from: "api_key", to: "apiKey"},
			{from: "channel_id", to: "channelId"},
			{from: "group_name", to: "groupName"},
			{from: "secret_access_key", to: "secretAccessKey"},
		},
		RemoveNames: []string{},
	}, data)
}

func translateApiDataToTf(data map[string]interface{}) map[string]interface{} {
	// We rename the read only fields the API returns on GET back to their
	// writable counterparts seen in the POST call.
	// Allowing them to be different here violates Terraform constructs and
	// makes them impossible to retrieve individually, although oddly enough
	// you can see them if you set the entire "data" map to an output variable.

	return modifyMap(ModifyMapArgs{
		RenameNames: []struct{ from, to string }{
			{from: "channelId", to: "channel_id"},
			{from: "groupName", to: "group_name"},
		},
		RemoveNames: []string{
			"accessKeyIdDisplay",
			"apiKeyDisplay",
			"secretAccessKeyDisplay",
			"tokenDisplay",
		},
	}, data)
}

func resourceIntegrationCreate(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	projectId := d.Get("project_id").(string)

	request := client.CreateIntegrationRequest{
		Data:        translateTfDataToApi(d.Get("data").(map[string]interface{})),
		Description: d.Get("description").(string),
	}

	resp, err := c.client.CreateIntegration(ctx, c.organizationId, projectId, request)
	if err != nil {
		return err
	}

	d.SetId(resp.Id)

	return resourceIntegrationRead(ctx, d, meta)
}

func resourceIntegrationRead(
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
	if err := d.Set("data", translateApiDataToTf(resp.Integration.Data)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceIntegrationDelete(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	var diags diag.Diagnostics

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
			return diags
		}
		time.Sleep(1.0)
	}
}

func resourceIntegrationUpdate(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	if !d.HasChanges("description", "data") {
		return resourceIntegrationRead(ctx, d, meta)
	}

	var desc *string
	desc = nil
	newDesc := d.Get("description").(string)
	if newDesc != "" {
		desc = &newDesc
	}

	var data *map[string]interface{}
	if d.HasChange("data") {
		switch v := d.Get("data").(type) {
		case nil:
			data = nil
		case map[string]interface{}:
			newData := translateTfDataToApi(v)
			data = &newData
		default:
			return diag.FromErr(fmt.Errorf("error - data was an unexpected type"))
		}
	} else {
		data = nil
	}

	request := client.UpdateIntegrationRequest{
		Data:        data,
		Description: desc,
	}

	orgId := c.organizationId
	projectId := d.Get("project_id").(string)
	integrationId := d.Id()

	if err := c.client.UpdateIntegration(ctx, orgId, projectId, integrationId, request); err != nil {
		return err
	}

	return resourceIntegrationRead(ctx, d, meta)
}
