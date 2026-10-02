package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var pagePropertyNumericID = regexp.MustCompile(`^[1-9][0-9]*$`)

func resourcePageProperty() *schema.Resource {
	return &schema.Resource{
		Create:   resourcePagePropertyCreate,
		Read:     resourcePagePropertyRead,
		Update:   resourcePagePropertyUpdate,
		Delete:   resourcePagePropertyDelete,
		Importer: &schema.ResourceImporter{StateContext: resourcePagePropertyImport},
		Schema: map[string]*schema.Schema{
			"page_id": {
				Type: schema.TypeString, Required: true, ForceNew: true,
				ValidateFunc: validation.StringMatch(pagePropertyNumericID, "must be a positive numeric page ID"),
				Description:  "ID of the page that owns this property.",
			},
			"key": {
				Type: schema.TypeString, Required: true, ForceNew: true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Property key, unique within the page.",
			},
			"value": {
				Type: schema.TypeString, Optional: true,
				ExactlyOneOf: []string{"value", "value_json"},
				Description:  "Plain string value. The provider performs JSON encoding.",
			},
			"value_json": {
				Type: schema.TypeString, Optional: true,
				ExactlyOneOf:          []string{"value", "value_json"},
				ValidateFunc:          validatePagePropertyJSON,
				DiffSuppressFunc:      pagePropertyDiffJSON,
				DiffSuppressOnRefresh: true,
				Description:           "JSON value, for example produced by jsonencode. Mutually exclusive with value.",
			},
			"adopt_existing": {
				Type: schema.TypeBool, Optional: true, Default: false,
				Description: "Allow creation to take ownership of an existing property with the same key. Destroy deletes it; its previous value is not restored.",
			},
			"property_id": {Type: schema.TypeString, Computed: true, Description: "Confluence property ID."},
			"version":     {Type: schema.TypeInt, Computed: true, Description: "Current property version, independent of the page version."},
		},
	}
}

func resourcePagePropertyCreate(d *schema.ResourceData, m interface{}) error {
	c := m.(*Client)
	pageID, key := d.Get("page_id").(string), d.Get("key").(string)
	value := pagePropertyValue(d)
	existing, err := c.FindPageProperty(pageID, key)
	if err != nil {
		return err
	}
	var property *PageProperty
	if existing != nil {
		if !d.Get("adopt_existing").(bool) {
			return fmt.Errorf("property %q already exists on page %s; import it using %s/%s or set adopt_existing = true", key, pageID, pageID, existing.ID)
		}
		if !pagePropertyNumericID.MatchString(existing.ID) {
			return fmt.Errorf("Confluence returned an invalid property ID")
		}
		property, err = c.UpdatePageProperty(pageID, existing.ID, key, value)
	} else {
		property, err = c.CreatePageProperty(pageID, key, value)
	}
	if err != nil {
		return err
	}
	if !pagePropertyNumericID.MatchString(property.ID) {
		return fmt.Errorf("Confluence returned an invalid property ID")
	}
	d.SetId(pageID + "/" + property.ID)
	return resourcePagePropertyRead(d, m)
}

func resourcePagePropertyRead(d *schema.ResourceData, m interface{}) error {
	pageID, propertyID, err := parsePagePropertyID(d.Id())
	if err != nil {
		return err
	}
	property, err := m.(*Client).GetPageProperty(pageID, propertyID)
	if isNotFound(err) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return err
	}
	if property.ID != propertyID || property.Version == nil || property.Version.Number < 1 || !json.Valid(property.Value) {
		return fmt.Errorf("Confluence returned an incomplete property %s", d.Id())
	}
	values := map[string]interface{}{
		"page_id": pageID, "property_id": propertyID, "key": property.Key,
		"version": property.Version.Number, "value": "", "value_json": "",
	}
	var stringValue string
	// A non-string remote value must remain visible as drift even when the
	// configured string is empty. Never coerce booleans, numbers or null to text.
	if !pagePropertyUsesJSON(d) && len(property.Value) > 0 && property.Value[0] == '"' && json.Unmarshal(property.Value, &stringValue) == nil {
		values["value"] = stringValue
	} else {
		values["value_json"] = string(property.Value)
	}
	for key, value := range values {
		if err := d.Set(key, value); err != nil {
			return err
		}
	}
	return nil
}

func resourcePagePropertyUpdate(d *schema.ResourceData, m interface{}) error {
	if d.HasChange("value") || d.HasChange("value_json") {
		pageID, propertyID, err := parsePagePropertyID(d.Id())
		if err != nil {
			return err
		}
		if _, err := m.(*Client).UpdatePageProperty(pageID, propertyID, d.Get("key").(string), pagePropertyValue(d)); err != nil {
			return err
		}
	}
	return resourcePagePropertyRead(d, m)
}

func resourcePagePropertyDelete(d *schema.ResourceData, m interface{}) error {
	pageID, propertyID, err := parsePagePropertyID(d.Id())
	if err != nil {
		return err
	}
	err = m.(*Client).DeletePageProperty(pageID, propertyID)
	if isNotFound(err) {
		return nil
	}
	return err
}

func resourcePagePropertyImport(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	pageID, propertyID, err := parsePagePropertyID(d.Id())
	if err != nil {
		return nil, err
	}
	if err := d.Set("page_id", pageID); err != nil {
		return nil, err
	}
	if err := d.Set("property_id", propertyID); err != nil {
		return nil, err
	}
	if err := d.Set("adopt_existing", false); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func parsePagePropertyID(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 || !pagePropertyNumericID.MatchString(parts[0]) || !pagePropertyNumericID.MatchString(parts[1]) {
		return "", "", fmt.Errorf("invalid page property ID %q: expected <page_id>/<property_id> with positive numeric IDs", id)
	}
	return parts[0], parts[1], nil
}

func pagePropertyUsesJSON(d *schema.ResourceData) bool {
	// Raw configuration distinguishes an explicitly empty string from an
	// omitted value and keeps the selected input mode during refresh.
	config := d.GetRawConfig()
	if config.IsKnown() && !config.IsNull() {
		return !config.GetAttr("value_json").IsNull()
	}
	// Imports have no configuration; preserve an existing JSON representation.
	return d.Get("value_json").(string) != ""
}

func pagePropertyValue(d *schema.ResourceData) json.RawMessage {
	if pagePropertyUsesJSON(d) {
		return json.RawMessage(d.Get("value_json").(string))
	}
	value, _ := json.Marshal(d.Get("value").(string))
	return value
}

func pagePropertyDiffJSON(_ string, old, new string, _ *schema.ResourceData) bool {
	left, err := normalizePagePropertyJSON(old)
	if err != nil {
		return false
	}
	right, err := normalizePagePropertyJSON(new)
	return err == nil && left == right
}

func validatePagePropertyJSON(value interface{}, key string) ([]string, []error) {
	text, ok := value.(string)
	if !ok || !json.Valid([]byte(text)) {
		return nil, []error{fmt.Errorf("%s must contain a valid JSON value", key)}
	}
	return nil, nil
}

func normalizePagePropertyJSON(value string) (string, error) {
	if !json.Valid([]byte(value)) {
		return "", fmt.Errorf("invalid JSON")
	}
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber() // Preserve integers larger than 2^53, including nested values.
	var decoded interface{}
	if err := decoder.Decode(&decoded); err != nil {
		return "", err
	}
	normalized, err := json.Marshal(decoded)
	return string(normalized), err
}
