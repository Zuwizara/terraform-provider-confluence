package confluence

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider returns the ResourceProvider for Confluence
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"cloud_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Atlassian Cloud ID used by the Confluence API gateway",
				DefaultFunc: schema.EnvDefaultFunc("CONFLUENCE_CLOUD_ID", nil),
			},
			"api_token": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Scoped Atlassian service-account API token",
				DefaultFunc: schema.EnvDefaultFunc("CONFLUENCE_API_TOKEN", nil),
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"confluence_space": dataSourceSpace(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"confluence_attachment": resourceAttachment(),
			"confluence_page":       resourcePage(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	return NewClient(&NewClientInput{
		cloudID:  d.Get("cloud_id").(string),
		apiToken: d.Get("api_token").(string),
	}), nil
}
