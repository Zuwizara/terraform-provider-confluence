package confluence

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func dataSourceSpace() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceSpaceRead,
		Schema: map[string]*schema.Schema{
			"key":         {Type: schema.TypeString, Required: true},
			"name":        {Type: schema.TypeString, Computed: true},
			"type":        {Type: schema.TypeString, Computed: true},
			"status":      {Type: schema.TypeString, Computed: true},
			"homepage_id": {Type: schema.TypeString, Computed: true},
		},
	}
}

func dataSourceSpaceRead(d *schema.ResourceData, m interface{}) error {
	space, err := m.(*Client).GetSpaceByKey(d.Get("key").(string))
	if err != nil {
		return err
	}
	d.SetId(space.ID)
	for key, value := range map[string]interface{}{
		"key": space.Key, "name": space.Name, "type": space.Type,
		"status": space.Status, "homepage_id": space.HomepageID,
	} {
		if err := d.Set(key, value); err != nil {
			return err
		}
	}
	return nil
}
