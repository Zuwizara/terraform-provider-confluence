---
layout: "confluence"
page_title: "Provider: Confluence"
sidebar_current: "docs-confluence-index"
description: |-
  The Confluence provider is used to interact with Confluence.
  It can automate the publishing of content and is often used to publish
  information about other resources created in terraform.
---

# Confluence Provider

The Confluence provider is used to interact with Confluence Cloud. Confluence
Data Center and Confluence Server are not supported. The provider needs to be
configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available data sources.

## Example Usage

```hcl
provider "confluence" {
	cloud_id = "00000000-0000-0000-0000-000000000000"
	api_token = var.confluence_api_token
}

data "confluence_space" "docs" {
	key = "MYSPACE"
}

resource "confluence_page" "default" {
	space_id = data.confluence_space.docs.id
	title    = "Example Page"
	body     = "<p>This page was built with Terraform</p>"
}
```

## Authentication

Use a scoped Atlassian service-account API token. The provider sends it as a
Bearer token to `https://api.atlassian.com/ex/confluence/{cloud_id}/wiki/api/v2`.

## Argument Reference

- `cloud_id` - (Required) Atlassian Cloud ID. It can also be set with
  `CONFLUENCE_CLOUD_ID`.
- `api_token` - (Required, Sensitive) Scoped Atlassian service-account token.
  It can also be set with `CONFLUENCE_API_TOKEN`.
