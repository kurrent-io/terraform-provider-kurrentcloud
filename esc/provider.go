package esc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/kurrent-io/terraform-provider-kurrentcloud/v2/client"
)

var defaultTokenStore = filepath.Join(os.Getenv("HOME"), ".esctf", "tokens")

func init() {
	schema.DescriptionKind = schema.StringMarkdown

	schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
		desc := s.Description
		if s.Default != nil {
			desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
		}
		if s.Deprecated != "" {
			desc += " " + s.Deprecated
		}
		return strings.TrimSpace(desc)
	}
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"url": {
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("ESC_URL", "https://api.eventstore.cloud"),
				},

				"token": {
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("ESC_TOKEN", ""),
					Sensitive:   true,
				},

				"organization_id": {
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("ESC_ORG_ID", ""),
				},

				"token_store": {
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("ESC_TOKEN_STORE", defaultTokenStore),
				},

				"identity_provider_url": {
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("ESC_IDENTITY_PROVIDER_URL", ""),
				},

				"client_id": {
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("ESC_CLIENT_ID", ""),
				},

				"client_secret": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("ESC_CLIENT_SECRET", ""),
					Sensitive:   true,
				},

				"identity_kit_url": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("ESC_IDENTITY_KIT_URL", ""),
				},
			},

			DataSourcesMap: map[string]*schema.Resource{
				"kurrentcloud_project": dataSourceProject(),
				"kurrentcloud_network": dataSourceNetwork(),

				// Deprecated names
				"eventstorecloud_project": dataSourceEventstorecloudProject(),
				"eventstorecloud_network": dataSourceEventstorecloudNetwork(),
			},

			ResourcesMap: map[string]*schema.Resource{
				// New preferred names
				"kurrentcloud_project":                           resourceProject(),
				"kurrentcloud_acl":                               resourceAcl(),
				"kurrentcloud_network":                           resourceNetwork(),
				"kurrentcloud_peering":                           resourcePeering(),
				"kurrentcloud_managed_cluster":                   resourceManagedCluster(),
				"kurrentcloud_managed_cluster_replicaset":        resourceManagedClusterReplicaset(),
				"kurrentcloud_scheduled_backup":                  resourceScheduledBackup(),
				"kurrentcloud_integration":                       resourceIntegration(),
				"kurrentcloud_integration_awscloudwatch_logs":    resourceIntegrationAwsCloudWatchLogs(),
				"kurrentcloud_integration_awscloudwatch_metrics": resourceIntegrationAwsCloudWatchMetrics(),

				// Deprecated names
				"eventstorecloud_project":                           resourceEventstorecloudProject(),
				"eventstorecloud_acl":                               resourceEventstorecloudAcl(),
				"eventstorecloud_network":                           resourceEventstorecloudNetwork(),
				"eventstorecloud_peering":                           resourceEventstorecloudPeering(),
				"eventstorecloud_managed_cluster":                   resourceEventstorecloudManagedCluster(),
				"eventstorecloud_scheduled_backup":                  resourceEventstorecloudScheduledBackup(),
				"eventstorecloud_integration":                       resourceEventstorecloudIntegration(),
				"eventstorecloud_integration_awscloudwatch_logs":    resourceEventstorecloudIntegrationAwsCloudWatchLogs(),
				"eventstorecloud_integration_awscloudwatch_metrics": resourceEventstorecloudIntegrationAwsCloudWatchMetrics(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

// This allows the Terraform validation to be bypassed. This can be useful if
// your using an older version of the plugin which cannot be upgraded for
// whatever reason and wish to use a newer allowed paramter value that the
// EventStore Cloud API supports
func ValidateWithByPass(f schema.SchemaValidateDiagFunc) schema.SchemaValidateDiagFunc {
	if v := os.Getenv("ESC_BYPASS_VALIDATION"); v != "" {
		return func(_ any, _ cty.Path) diag.Diagnostics {
			return diag.Diagnostics{}
		}
	} else {
		return f
	}
}

func configure(
	version string,
	p *schema.Provider,
) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		// Set Terraform version for telemetry from the provider instance
		if p.TerraformVersion != "" {
			client.TerraformVersion = p.TerraformVersion
		}

		config := &client.Config{
			URL:                 d.Get("url").(string),
			RefreshToken:        d.Get("token").(string),
			TokenStore:          d.Get("token_store").(string),
			IdentityProviderURL: d.Get("identity_provider_url").(string),
			IdentityKitURL:      d.Get("identity_kit_url").(string),
			ClientID:            d.Get("client_id").(string),
			ClientSecret:        d.Get("client_secret").(string),
		}

		c, err := client.New(config)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		return &providerContext{
			organizationId: d.Get("organization_id").(string),
			client:         c,
		}, nil
	}
}

type providerContext struct {
	organizationId string
	client         *client.Client
}

var validProviders = []string{"aws", "gcp", "azure"}

// Note: Versions < 22.10 and 23.6 are no longer supported
var (
	validTopologies       = []string{"single-node", "three-node-multi-zone"}
	validInstanceTypes    = []string{"F1", "C4", "M8", "M16", "M32", "M64", "M128"}
	validDiskTypes        = []string{"GP3", "SSD", "PREMIUM-SSD-LRS"}
	validProjectionLevels = []string{"off", "system", "user"}
)
