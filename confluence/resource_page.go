package confluence

import (
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourcePage() *schema.Resource {
	return &schema.Resource{
		Create: resourcePageCreate,
		Read:   resourcePageRead,
		Update: resourcePageUpdate,
		Delete: resourcePageDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"space_id": {Type: schema.TypeString, Required: true, ForceNew: true},
			"title": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(1, 255),
			},
			"body": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: resourcePageDiffBody,
			},
			"parent_id": {Type: schema.TypeString, Optional: true, Computed: true},
			"version":   {Type: schema.TypeInt, Computed: true},
			"url":       {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourcePageCreate(d *schema.ResourceData, m interface{}) error {
	page, err := m.(*Client).CreatePage(pageFromResourceData(d))
	if err != nil {
		return err
	}
	d.SetId(page.ID)
	return resourcePageRead(d, m)
}

func resourcePageRead(d *schema.ResourceData, m interface{}) error {
	page, err := m.(*Client).GetPage(d.Id())
	if isNotFound(err) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return err
	}
	if page.Status == "trashed" {
		d.SetId("")
		return nil
	}
	return updateResourceDataFromPage(d, page)
}

func resourcePageUpdate(d *schema.ResourceData, m interface{}) error {
	if _, err := m.(*Client).UpdatePage(pageFromResourceData(d)); err != nil {
		return err
	}
	return resourcePageRead(d, m)
}

func resourcePageDelete(d *schema.ResourceData, m interface{}) error {
	return m.(*Client).DeletePage(d.Id())
}

func pageFromResourceData(d *schema.ResourceData) *Page {
	return &Page{
		ID:       d.Id(),
		SpaceID:  d.Get("space_id").(string),
		ParentID: d.Get("parent_id").(string),
		Status:   "current",
		Title:    d.Get("title").(string),
		Body: &PageBody{
			Representation: "storage",
			Value:          d.Get("body").(string),
		},
	}
}

func updateResourceDataFromPage(d *schema.ResourceData, page *Page) error {
	body := ""
	if page.Body != nil {
		if page.Body.Storage != nil {
			body = page.Body.Storage.Value
		} else {
			body = page.Body.Value
		}
	}
	version := 0
	if page.Version != nil {
		version = page.Version.Number
	}
	pageURL := ""
	if page.Links != nil && page.Links.Base != "" && page.Links.WebUI != "" {
		pageURL = strings.TrimRight(page.Links.Base, "/") + "/" + strings.TrimLeft(page.Links.WebUI, "/")
	}
	for key, value := range map[string]interface{}{
		"space_id": page.SpaceID, "parent_id": page.ParentID, "title": page.Title,
		"body": body, "version": version, "url": pageURL,
	} {
		if err := d.Set(key, value); err != nil {
			return err
		}
	}
	return nil
}

func resourcePageDiffBody(_ string, old, new string, _ *schema.ResourceData) bool {
	macroID := regexp.MustCompile(` *ac:macro-id="[\w\d\-]*" *`)
	return macroID.ReplaceAllString(strings.TrimSpace(old), "") == macroID.ReplaceAllString(strings.TrimSpace(new), "")
}
