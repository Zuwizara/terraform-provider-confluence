terraform {
  required_providers {
    confluence = {
      source = "zuwizara/confluence"
    }
  }
}

# Uses CONFLUENCE_CLOUD_ID and CONFLUENCE_API_TOKEN.
provider "confluence" {}

variable "space_key" {
  type        = string
  description = "Key of the Confluence space for the example page."
}

data "confluence_space" "example" {
  key = var.space_key
}

resource "confluence_page" "example" {
  space_id = data.confluence_space.example.id
  title    = "Page properties example"
  body     = "<p>Managed by Terraform with full-width layout.</p>"
}

resource "confluence_page_property" "wide_layout" {
  for_each = toset([
    "content-appearance-draft",
    "content-appearance-published",
  ])

  page_id        = confluence_page.example.id
  key            = each.value
  value          = "full-width"
  adopt_existing = true
}

resource "confluence_page_property" "metadata" {
  page_id = confluence_page.example.id
  key     = "infrastructure"
  value_json = jsonencode({
    owner   = "platform"
    enabled = true
    regions = ["eu-central-1"]
  })
}
