package esc

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/kurrent-io/terraform-provider-kurrentcloud/v2/client"
)

func dataSourceProject() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves data for an existing `Project` resource",

		ReadContext: dataSourceProjectRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceEventstorecloudProject() *schema.Resource {
	return &schema.Resource{
		Description:        "Retrieves data for an existing `Project` resource",
		DeprecationMessage: "Use kurrentcloud_project instead. eventstorecloud_project will be removed in v3.0.0",

		ReadContext: dataSourceProjectRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceProjectRead(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	resp, err := c.client.ProjectList(ctx, &client.ListProjectsRequest{
		OrganizationID: c.organizationId,
	})
	if err != nil {
		return err
	}

	if len(resp.Projects) == 0 {
		return diag.Errorf("There are no projects in organization %s", c.organizationId)
	}

	var found []*client.Project
	desiredName := d.Get("name").(string)
	for _, project := range resp.Projects {
		if project.Name == desiredName {
			found = append(found, &project)
			break
		}
	}

	if len(found) == 0 {
		return diag.Errorf(
			"Project %s was not found in organization %s",
			desiredName,
			c.organizationId,
		)
	}
	if len(found) > 1 {
		return diag.Errorf(
			"There are more than one project with name %s in organization %s",
			desiredName,
			c.organizationId,
		)
	}

	d.SetId(found[0].ProjectID)

	return nil
}
