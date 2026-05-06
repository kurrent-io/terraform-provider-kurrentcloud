package esc

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/kurrent-io/terraform-provider-kurrentcloud/v2/client"
)

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "Manages VPC (network) resources in Kurrent Cloud",

		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		UpdateContext: resourceNetworkUpdate,
		DeleteContext: resourceNetworkDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceImport,
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Description: "Project ID",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"resource_provider": {
				Description: "Cloud Provider in which to provision the network.",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
				ValidateDiagFunc: ValidateWithByPass(
					validation.ToDiagFunc(validation.StringInSlice(validProviders, true)),
				),
				StateFunc: func(val interface{}) string {
					// Normalize to lower case
					return strings.ToLower(val.(string))
				},
			},
			"region": {
				Description: "Provider region in which to provision the network",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"cidr_block": {
				Description:  "Address space of the network in CIDR block notation",
				Required:     false,
				ForceNew:     true,
				Default:      "",
				Type:         schema.TypeString,
				ValidateFunc: validation.IsCIDRNetwork(8, 24),
				Optional:     true,
			},
			"name": {
				Description: "Human-friendly name for the network",
				Type:        schema.TypeString,
				Required:    true,
			},
			"public_access": {
				Description: "Whether the network is able to be accessed from the public internet",
				Type:        schema.TypeBool,
				Default:     false,
				Required:    false,
				ForceNew:    true,
				Optional:    true,
			},
		},
	}
}

func resourceEventstorecloudNetwork() *schema.Resource {
	return &schema.Resource{
		Description:        "Manages VPC (network) resources in Kurrent Cloud",
		DeprecationMessage: "Use kurrentcloud_network instead. eventstorecloud_network will be removed in v3.0.0",

		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		UpdateContext: resourceNetworkUpdate,
		DeleteContext: resourceNetworkDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceImport,
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Description: "Project ID",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"resource_provider": {
				Description: "Cloud Provider in which to provision the network.",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
				ValidateDiagFunc: ValidateWithByPass(
					validation.ToDiagFunc(validation.StringInSlice(validProviders, true)),
				),
				StateFunc: func(val interface{}) string {
					// Normalize to lower case
					return strings.ToLower(val.(string))
				},
			},
			"region": {
				Description: "Provider region in which to provision the network",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"cidr_block": {
				Description:  "Address space of the network in CIDR block notation",
				Required:     false,
				ForceNew:     true,
				Default:      "",
				Type:         schema.TypeString,
				ValidateFunc: validation.IsCIDRNetwork(8, 24),
				Optional:     true,
			},
			"name": {
				Description: "Human-friendly name for the network",
				Type:        schema.TypeString,
				Required:    true,
			},
			"public_access": {
				Description: "Whether the network is able to be accessed from the public internet",
				Type:        schema.TypeBool,
				Default:     false,
				Required:    false,
				ForceNew:    true,
				Optional:    true,
			},
		},
	}
}

func resourceNetworkCreate(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	projectId := d.Get("project_id").(string)

	request := &client.CreateNetworkRequest{
		OrganizationID:   c.organizationId,
		ProjectID:        projectId,
		ResourceProvider: d.Get("resource_provider").(string),
		CidrBlock:        d.Get("cidr_block").(string),
		Name:             d.Get("name").(string),
		Region:           d.Get("region").(string),
		PublicAccess:     d.Get("public_access").(bool),
	}

	if request.PublicAccess && request.CidrBlock != "" {
		return diag.Errorf(
			"Error: network resources with \"public_access\" set to true can not set \"cidr_block\".",
		)
	}
	if !request.PublicAccess && request.CidrBlock == "" {
		return diag.Errorf(
			"Error: network resources with \"public_access\" set to false must set \"cidr_block\".",
		)
	}

	resp, err := c.client.NetworkCreate(ctx, request)
	if err != nil {
		return err
	}

	d.SetId(resp.NetworkID)

	if err := c.client.NetworkWaitForState(ctx, &client.WaitForNetworkStateRequest{
		OrganizationID: c.organizationId,
		ProjectID:      projectId,
		NetworkID:      resp.NetworkID,
		State:          "available",
	}); err != nil {
		return err
	}

	return resourceNetworkRead(ctx, d, meta)
}

func resourceNetworkUpdate(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	if d.HasChange("name") {
		projectId := d.Get("project_id").(string)
		networkId := d.Id()

		request := &client.UpdateNetworkRequest{
			OrganizationID: c.organizationId,
			ProjectID:      projectId,
			NetworkID:      networkId,
			Name:           d.Get("name").(string),
		}

		err := c.client.NetworkUpdate(ctx, request)
		if err != nil {
			return err
		}
	}

	return resourceProjectRead(ctx, d, meta)
}

func resourceNetworkRead(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	var diags diag.Diagnostics

	projectId := d.Get("project_id").(string)
	networkId := d.Id()

	request := &client.GetNetworkRequest{
		OrganizationID: c.organizationId,
		ProjectID:      projectId,
		NetworkID:      networkId,
	}

	resp, err := c.client.NetworkGet(ctx, request)
	if err != nil {
		return diag.Errorf("Internal Server Error, try again later")
	}
	if resp.Network.Status == client.StateDeleted {
		d.SetId("")
		return nil
	}

	// set the returned cidr block to blank so that terraform will think it
	// hasn't changed from its initial value
	if resp.Network.PublicAccess {
		resp.Network.CIDRBlock = ""
	}

	if err := d.Set("project_id", resp.Network.ProjectID); err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("Unable to set project_id: %w", err))...)
	}
	if err := d.Set("resource_provider", resp.Network.Provider); err != nil {
		diags = append(
			diags,
			diag.FromErr(fmt.Errorf("Unable to set resource_provider: %w", err))...)
	}
	if err := d.Set("region", resp.Network.Region); err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("Unable to set region: %w", err))...)
	}
	if err := d.Set("cidr_block", resp.Network.CIDRBlock); err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("Unable to set cidr_block: %w", err))...)
	}
	if err := d.Set("name", resp.Network.Name); err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("Unable to set name: %w", err))...)
	}
	if err := d.Set("public_access", resp.Network.PublicAccess); err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("Unable to set public_access: %w", err))...)
	}
	return diags
}

func resourceNetworkDelete(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	var diags diag.Diagnostics

	projectId := d.Get("project_id").(string)
	networkId := d.Id()

	request := &client.DeleteNetworkRequest{
		OrganizationID: c.organizationId,
		ProjectID:      projectId,
		NetworkID:      networkId,
	}

	if err := c.client.NetworkDelete(ctx, request); err != nil {
		return err
	}

	if err := c.client.NetworkWaitForState(ctx, &client.WaitForNetworkStateRequest{
		OrganizationID: c.organizationId,
		ProjectID:      projectId,
		NetworkID:      networkId,
		State:          "deleted",
	}); err != nil {
		return err
	}

	return diags
}
