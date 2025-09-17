package esc

import (
	"context"
	"fmt"
	"path"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/kurrent-io/terraform-provider-kurrentcloud/client"
)

func resourcePeering() *schema.Resource {
	return &schema.Resource{
		Description:        "Manages peering connections between Kurrent Cloud VPCs and customer own VPCs",
		DeprecationMessage: "Use kurrentcloud_peering instead. eventstorecloud_peering will be removed in v3.0.0",

		CreateContext: resourcePeeringCreate,
		ReadContext:   resourcePeeringRead,
		UpdateContext: resourcePeeringUpdate,
		DeleteContext: resourcePeeringDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceImport,
		},

		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Description: "Project ID",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"network_id": {
				Description: "Network ID",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"peer_resource_provider": {
				Description:  "Cloud Provider in which the target network exists",
				Required:     true,
				ForceNew:     true,
				Type:         schema.TypeString,
				ValidateFunc: validation.StringInSlice(validProviders, true),
			},
			"peer_network_region": {
				Description: "Provider region in which to the peer network exists",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"peer_account_id": {
				Description: "Account identifier in which to the peer network exists",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"peer_network_id": {
				Description: "Network identifier of the peer network exists",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"name": {
				Description: "Human-friendly name for the network",
				Type:        schema.TypeString,
				Required:    true,
			},
			"routes": {
				Description: "Routes to create from the Event Store network to the peer network",
				Type:        schema.TypeSet,
				ForceNew:    true,
				Required:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsCIDRNetwork(8, 28),
				},
				Set: schema.HashString,
			},

			"provider_metadata": {
				Description: "Metadata about the remote end of the peering connection",
				Type:        schema.TypeMap,
				Computed:    true,
				Elem: &schema.Schema{
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},

		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    cty.Object(map[string]cty.Type{}),
				Upgrade: upgrade1_5_6,
			},
		},
	}
}

func upgrade1_5_6(
	_ context.Context,
	state map[string]interface{},
	meta interface{},
) (map[string]interface{}, error) {
	// In version 1.5.5 / 1.5.6 we accidentally made the field
	// `provider_metadata` into a set of maps instead of a map. Changing the
	// type back to a map means users who have projects using 1.5.6 will get an
	// unresolvable situation that even updating their code can't fix. So we
	// need to check to see if this item is a map / list, and if so just change
	// it into it's first (and only) item.

	if providerMetadata, exists := state["provider_metadata"]; exists {
		providerMetadataList, ok := providerMetadata.([]interface{})
		if ok {
			// Because the field was calculated there is no risk the element count will
			// ever be != 1. If it somehow is though making it into an empty map
			// should also be safe.
			if len(providerMetadataList) == 1 {
				state["provider_metadata"] = providerMetadataList[0]
			} else {
				state["provider_metadata"] = map[string]interface{}{}
			}
		}
	}
	return state, nil
}

func resourcePeeringSetProviderMetadata(
	d *schema.ResourceData,
	provider string,
	metadata map[string]string,
) diag.Diagnostics {
	providerPeeringMetadata := map[string]interface{}{}

	var diags diag.Diagnostics

	switch provider {
	case "aws":
		if val, hasVal := metadata["peeringLinkId"]; hasVal {
			providerPeeringMetadata["aws_peering_link_id"] = val
		} else {
			diags = append(diags, Warnof("AWS peering link missing remote peering link identifier")...)
		}
	case "gcp":
		projectId, hasProjectId := metadata["projectId"]
		if hasProjectId {
			providerPeeringMetadata["gcp_project_id"] = projectId
		} else {
			diags = append(diags, diag.Errorf("GCP peering link missing remote peering link project identifier")...)
		}
		networkName, hasNetworkName := metadata["networkId"]
		if hasNetworkName {
			providerPeeringMetadata["gcp_network_name"] = networkName
		} else {
			diags = append(diags, diag.Errorf("GCP peering link missing remote peering link network identifier")...)
		}
		providerPeeringMetadata["gcp_network_id"] = path.Join(
			"projects",
			projectId,
			"global",
			"networks",
			networkName)
	case "azure":
		break
	default:
		diags = append(
			diags,
			diag.Errorf("Unknown provider %q from Event Store Cloud API", provider)...)
	}

	if err := d.Set("provider_metadata", providerPeeringMetadata); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}

func resourcePeeringCreate(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	projectId := d.Get("project_id").(string)

	routesSet := d.Get("routes").(*schema.Set)
	routes := make([]string, routesSet.Len())
	for i, route := range routesSet.List() {
		routes[i] = route.(string)
	}

	request := &client.CreatePeeringRequest{
		OrganizationID:        c.organizationId,
		ProjectID:             projectId,
		NetworkId:             d.Get("network_id").(string),
		Name:                  d.Get("name").(string),
		PeerAccountIdentifier: d.Get("peer_account_id").(string),
		PeerNetworkIdentifier: d.Get("peer_network_id").(string),
		PeerNetworkRegion:     d.Get("peer_network_region").(string),
		Routes:                routes,
	}

	resp, err := c.client.PeeringCreate(ctx, request)
	if err != nil {
		return err
	}

	d.SetId(resp.PeeringID)

	peering, err := c.client.PeeringWaitForState(ctx, &client.WaitForPeeringStateRequest{
		OrganizationID: c.organizationId,
		ProjectID:      projectId,
		PeeringID:      resp.PeeringID,
		State:          "initiated",
	})
	if err != nil {
		return err
	}

	if err := resourcePeeringSetProviderMetadata(d, peering.Provider, peering.ProviderPeeringMetadata); err != nil {
		return err
	}

	return resourcePeeringReadCheckDefunct(ctx, d, meta)
}

func resourcePeeringUpdate(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	if d.HasChange("name") {
		request := &client.UpdatePeeringRequest{
			OrganizationID: c.organizationId,
			ProjectID:      d.Get("project_id").(string),
			PeeringID:      d.Id(),
			Name:           d.Get("name").(string),
		}

		if err := c.client.PeeringUpdate(ctx, request); err != nil {
			return err
		}
	}

	return resourcePeeringRead(ctx, d, meta)
}

func resourcePeeringRead(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	return resourcePeeringReadWithCheck(ctx, d, meta, false)
}

func resourcePeeringReadCheckDefunct(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	return resourcePeeringReadWithCheck(ctx, d, meta, true)
}

func resourcePeeringReadWithCheck(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
	errorOnDefunct bool,
) diag.Diagnostics {
	c := meta.(*providerContext)

	var diags diag.Diagnostics

	projectId := d.Get("project_id").(string)
	peeringId := d.Id()

	request := &client.GetPeeringRequest{
		OrganizationID: c.organizationId,
		ProjectID:      projectId,
		PeeringID:      peeringId,
	}

	resp, err := c.client.PeeringGet(ctx, request)
	if err != nil {
		return diag.Errorf("Internal Server Error, try again later")
	}
	if resp.Peering.Status == client.StateDeleted {
		d.SetId("")
		return nil
	}

	if errorOnDefunct && resp.Peering.Status == client.StateDefunct {
		diags = append(
			diags,
			diag.FromErr(
				fmt.Errorf(
					"peering %s (in project %s) entered defunct state, check peering configuration",
					peeringId,
					projectId,
				),
			)...)
		return diags
	}

	if err := d.Set("project_id", resp.Peering.ProjectID); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("network_id", resp.Peering.NetworkID); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("peer_resource_provider", resp.Peering.Provider); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("peer_network_region", resp.Peering.PeerNetworkRegion); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("peer_account_id", resp.Peering.PeerAccountIdentifier); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("peer_network_id", resp.Peering.PeerNetworkIdentifier); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("name", resp.Peering.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("routes", resp.Peering.Routes); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := resourcePeeringSetProviderMetadata(d, resp.Peering.Provider, resp.Peering.ProviderPeeringMetadata); err != nil {
		diags = append(diags, err...)
	}

	return diags
}

func resourcePeeringDelete(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) diag.Diagnostics {
	c := meta.(*providerContext)

	projectId := d.Get("project_id").(string)
	peeringId := d.Id()

	request := &client.DeletePeeringRequest{
		OrganizationID: c.organizationId,
		ProjectID:      projectId,
		PeeringID:      peeringId,
	}

	if err := c.client.PeeringDelete(ctx, request); err != nil {
		return err
	}

	peering, err := c.client.PeeringWaitForState(ctx, &client.WaitForPeeringStateRequest{
		OrganizationID: c.organizationId,
		ProjectID:      projectId,
		PeeringID:      peeringId,
		State:          "deleted",
	})
	if peering.Status != "deleted" {
		return diag.Errorf("Peering wait for status returned, but the state is still not correct")
	}
	return err
}

func Warnof(format string, a ...interface{}) diag.Diagnostics {
	return diag.Diagnostics{
		diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf(format, a...),
		},
	}
}
