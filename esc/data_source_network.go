package esc

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/kurrent-io/terraform-provider-kurrentcloud/client"
)

func dataSourceNetwork() *schema.Resource {
	return &schema.Resource{
		Description:        "Retrieves data for an existing `Network` resource",
		DeprecationMessage: "Use kurrentcloud_network instead. eventstorecloud_network will be removed in v3.0.0",

		ReadContext: dataSourceNetworkRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"project_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"resource_provider": {
				Description: "Cloud Provider in which to provision the network.",
				Required:    false,
				ForceNew:    false,
				Computed:    true,
				Type:        schema.TypeString,
			},
			"region": {
				Description: "Provider region in which to provision the network",
				Required:    false,
				ForceNew:    false,
				Computed:    true,
				Type:        schema.TypeString,
			},
			"cidr_block": {
				Description: "Address space of the network in CIDR block notation",
				Required:    false,
				ForceNew:    false,
				Computed:    true,
				Type:        schema.TypeString,
			},
		},
	}
}

func dataSourceNetworkRead(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	projectID := d.Get("project_id").(string)

	resp, err := c.client.NetworkList(ctx, &client.ListNetworksRequest{
		OrganizationID: c.organizationId,
		ProjectID:      projectID,
	})
	if err != nil {
		return err
	}

	if len(resp.Networks) == 0 {
		return diag.Errorf("There are no networks in project %s", projectID)
	}

	var found *client.Network
	multipleNetworksFound := false
	count := 0

	desiredName := d.Get("name").(string)
	for _, network := range resp.Networks {
		if network.Name == desiredName && network.Status == "available" {
			count++
			if count > 1 {
				multipleNetworksFound = true
				break
			}
			found = &network
		}
	}

	if multipleNetworksFound {
		return diag.Errorf(
			"Error: Multiple networks with the same name '%s' were found. Please specify a more unique name or check your existing resources.",
			desiredName,
		)
	}

	if found == nil {
		return diag.Errorf("Network %s was not found in project %s", desiredName, projectID)
	}

	d.SetId(found.NetworkID)
	if err := d.Set("cidr_block", found.CIDRBlock); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("region", found.Region); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("resource_provider", found.Provider); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
