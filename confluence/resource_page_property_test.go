package confluence

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Exercise Terraform's actual refresh, planning, import and apply behavior
// without requiring cloud credentials or making external API calls.
func TestPagePropertyTerraformLifecycle(t *testing.T) {
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("Terraform CLI is required for the local lifecycle test")
	}
	var mu sync.Mutex
	var property *PageProperty
	nextID := 99
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		collection := "/ex/confluence/cloud-123/wiki/api/v2/pages/42/properties"
		if r.URL.Path != collection && !strings.HasPrefix(r.URL.Path, collection+"/") {
			t.Errorf("unexpected request path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path == collection {
				results := []*PageProperty{}
				if property != nil && r.URL.Query().Get("key") == property.Key {
					results = append(results, property)
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
			} else if property == nil || r.URL.Path != collection+"/"+property.ID {
				http.NotFound(w, r)
			} else {
				_ = json.NewEncoder(w).Encode(property)
			}
		case http.MethodPost, http.MethodPut:
			var request PageProperty
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			if r.Method == http.MethodPost {
				if property != nil {
					w.WriteHeader(400)
					return
				}
				request.ID = fmt.Sprint(nextID)
				nextID++
				request.Version = &Version{Number: 1}
			} else {
				if property == nil || request.Version == nil || request.Version.Number != property.Version.Number+1 {
					w.WriteHeader(409)
					return
				}
				request.ID = property.ID
			}
			property = &request
			_ = json.NewEncoder(w).Encode(property)
		case http.MethodDelete:
			property = nil
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()
	client := testClient(t, server)
	config := func(value string) string {
		return `provider "confluence" {
  cloud_id = "test"
  api_token = "test"
}
resource "confluence_page_property" "test" {
  page_id = "42"
  key = "layout"
  ` + value + "\n}\n"
	}
	name := "confluence_page_property.test"
	check := func(key, value string) resource.TestCheckFunc {
		return resource.TestCheckResourceAttr(name, key, value)
	}
	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"confluence": func() (*schema.Provider, error) {
				p := Provider()
				p.ConfigureFunc = func(_ *schema.ResourceData) (interface{}, error) { return client, nil }
				return p, nil
			},
		},
		CheckDestroy: func(_ *terraform.State) error {
			mu.Lock()
			defer mu.Unlock()
			if property != nil {
				return fmt.Errorf("property was not deleted")
			}
			return nil
		},
		Steps: []resource.TestStep{
			{Config: config(`value = "full-width"`), Check: check("value", "full-width")},
			{ResourceName: name, ImportState: true, ImportStateVerify: true},
			{Config: config(`value = ""`), Check: check("value", "")},
			// A non-string replacing an empty string must still trigger a repair.
			{PreConfig: func() { mu.Lock(); property.Value = json.RawMessage(`true`); property.Version.Number++; mu.Unlock() },
				Config: config(`value = ""`), Check: check("value", "")},
			{Config: config(`value_json = "true"`), Check: check("value_json", "true")},
			{ResourceName: name, ImportState: true, ImportStateVerify: true},
			{Config: config(`value_json = "null"`), Check: check("value_json", "null")},
			{Config: config(`value_json = "{\"b\":true,\"a\":9007199254740993}"`), Check: check("value_json", `{"b":true,"a":9007199254740993}`)},
			{Config: config(`value_json = "{ \"a\": 9007199254740993, \"b\": true }"`), PlanOnly: true},
			{Config: config(`value_json = "\"full-width\""`), Check: check("value_json", `"full-width"`)},
			{Config: config(`value = "full-width"`), Check: check("value", "full-width")},
			{PreConfig: func() {
				mu.Lock()
				property.Value = json.RawMessage(`"changed"`)
				property.Version.Number++
				mu.Unlock()
			},
				Config: config(`value = "full-width"`), Check: check("value", "full-width")},
			{PreConfig: func() { mu.Lock(); property = nil; mu.Unlock() },
				Config: config(`value = "full-width"`), Check: check("property_id", "100")},
		},
	})
}

func TestAccConfluencePageProperty(t *testing.T) {
	name := "confluence_page_property.test"
	pageTitle := acctest.RandomWithPrefix("terraform-property-test")
	config := func(value string) string {
		return testAccConfluencePageConfig(pageTitle, "<p>Property test</p>") + fmt.Sprintf(`
resource "confluence_page_property" "test" {
  page_id = confluence_page.default.id
  key = "terraform-provider-test"
  value = %q
}
`, value)
	}
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) }, Providers: testAccProviders,
		CheckDestroy: testAccCheckConfluenceDestroy,
		Steps: []resource.TestStep{
			{Config: config("initial"), Check: resource.TestCheckResourceAttr(name, "value", "initial")},
			{Config: config("updated"), Check: resource.TestCheckResourceAttr(name, "value", "updated")},
			{ResourceName: name, ImportState: true, ImportStateVerify: true},
		},
	})
}
