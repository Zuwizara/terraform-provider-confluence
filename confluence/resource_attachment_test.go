package confluence

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccConfluenceAttachment_Created(t *testing.T) {
	rName := acctest.RandomWithPrefix("resource-attachment-test")
	resourceName := "confluence_attachment.default"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConfluenceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckConfluenceAttachmentConfigRequired(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConfluenceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "title", "file.txt"),
					resource.TestCheckResourceAttr(resourceName, "data", rName),
					resource.TestCheckResourceAttr(resourceName, "version", "1"),
				),
			},
		},
	})
}

func testAccCheckConfluenceAttachmentConfigRequired(rName string) string {
	time.Sleep(time.Second)
	return fmt.Sprintf(`
data "confluence_space" "test" {
  key = %q
}

resource "confluence_page" "default" {
  space_id = data.confluence_space.test.id
  title    = %q
  body     = "Original value"
}

resource "confluence_attachment" "default" {
  title = "file.txt"
  data  = %q
  page_id = confluence_page.default.id
}
`, os.Getenv("CONFLUENCE_SPACE"), rName, rName)
}
