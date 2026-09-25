package confluence

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"confluence": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = Provider()
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("CONFLUENCE_CLOUD_ID"); v == "" {
		t.Fatal("CONFLUENCE_CLOUD_ID must be set for acceptance tests")
	}
	if v := os.Getenv("CONFLUENCE_API_TOKEN"); v == "" {
		t.Fatal("CONFLUENCE_API_TOKEN must be set for acceptance tests")
	}
	if v := os.Getenv("CONFLUENCE_SPACE"); v == "" {
		t.Fatal("CONFLUENCE_SPACE must be set for acceptance tests")
	}
}

func TestProviderCloudV2Schema(t *testing.T) {
	provider := Provider()
	if _, ok := provider.Schema["cloud_id"]; !ok {
		t.Fatal("cloud_id is not configured")
	}
	token, ok := provider.Schema["api_token"]
	if !ok || !token.Sensitive {
		t.Fatal("api_token must be configured as sensitive")
	}
	if _, ok := provider.DataSourcesMap["confluence_space"]; !ok {
		t.Fatal("confluence_space data source is not registered")
	}
	if _, ok := provider.ResourcesMap["confluence_page"]; !ok {
		t.Fatal("confluence_page resource is not registered")
	}
	if _, ok := provider.ResourcesMap["confluence_content"]; ok {
		t.Fatal("legacy confluence_content resource must not be registered")
	}
	if _, ok := provider.ResourcesMap["confluence_file_attachment"]; ok {
		t.Fatal("attachments must be represented by the single confluence_attachment resource")
	}
}
func testAccCheckConfluenceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)
	return confluenceDestroyHelper(s, client)
}

func testAccCheckConfluenceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Client)
		return confluenceExistsHelper(s, client)
	}
}

func confluenceDestroyHelper(s *terraform.State, client *Client) error {
	for _, r := range s.RootModule().Resources {
		id := r.Primary.ID
		switch r.Type {
		case "confluence_page":
			page, err := client.GetPage(id)
			if err == nil && page.Status != "trashed" {
				return fmt.Errorf("page still exists with status %q, id: %s", page.Status, id)
			}
		case "confluence_attachment":
			_, err := client.GetAttachment(id)
			if err == nil {
				return fmt.Errorf("Attachment still exists. id: %s", id)
			}
		default:
			return fmt.Errorf("Unknown resource: type = %s, id = %s", r.Type, id)
		}
	}
	return nil
}

func confluenceExistsHelper(s *terraform.State, client *Client) error {
	for _, r := range s.RootModule().Resources {
		id := r.Primary.ID
		switch r.Type {
		case "confluence_page":
			page, err := client.GetPage(id)
			if err != nil {
				return fmt.Errorf("received an error retrieving page: %s", err)
			}
			if page.Status == "trashed" {
				return fmt.Errorf("page %s is trashed", id)
			}
		case "confluence_attachment":
			_, err := client.GetAttachment(id)
			if err != nil {
				return fmt.Errorf("Received an error retrieving attachment %s", err)
			}
		default:
			return fmt.Errorf("Unknown resource: type = %s, id = %s", r.Type, id)
		}
	}
	return nil
}
