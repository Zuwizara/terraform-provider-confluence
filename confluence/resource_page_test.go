package confluence

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccConfluencePageUpdated(t *testing.T) {
	name := acctest.RandomWithPrefix("terraform-page-test")
	resourceName := "confluence_page.default"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConfluenceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfluencePageConfig(name, "Original value"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConfluenceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "title", name),
					resource.TestCheckResourceAttr(resourceName, "body", "Original value"),
					resource.TestCheckResourceAttr(resourceName, "version", "1"),
				),
			},
			{
				Config: testAccConfluencePageConfig(name, "Updated value"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConfluenceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "body", "Updated value"),
					resource.TestCheckResourceAttr(resourceName, "version", "2"),
				),
			},
		},
	})
}

func TestAccConfluencePageRecreateAfterTrash(t *testing.T) {
	name := acctest.RandomWithPrefix("terraform-page-recreate-test")
	resourceName := "confluence_page.default"
	var originalID string
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccConfluencePageConfig(name, "Original value"),
				Check: func(state *terraform.State) error {
					originalID = state.RootModule().Resources[resourceName].Primary.ID
					return nil
				},
			},
			{Config: testAccConfluenceSpaceOnlyConfig()},
			{
				Config: testAccConfluencePageConfig(name, "Recreated value"),
				Check: func(state *terraform.State) error {
					recreatedID := state.RootModule().Resources[resourceName].Primary.ID
					if recreatedID == originalID {
						return fmt.Errorf("expected a new page ID after trash, still %s", recreatedID)
					}
					return nil
				},
			},
		},
	})
}

func testAccConfluencePageConfig(name, body string) string {
	return fmt.Sprintf(`
data "confluence_space" "test" {
  key = %q
}

resource "confluence_page" "default" {
  space_id = data.confluence_space.test.id
  title    = %q
  body     = %q
}
`, os.Getenv("CONFLUENCE_SPACE"), name, body)
}

func testAccConfluenceSpaceOnlyConfig() string {
	return fmt.Sprintf(`
data "confluence_space" "test" {
  key = %q
}
`, os.Getenv("CONFLUENCE_SPACE"))
}
