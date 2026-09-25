---
layout: "confluence"
page_title: "Confluence: confluence_attachment"
sidebar_current: "docs-confluence-resource-attachment"
description: |-
  Provides attachments in Confluence
---

# confluence_attachment

Adds an attachment to a Confluence page. The content can be provided inline
with `data` or read from a local file with `file_path`.

## Example Usage

```hcl
data "confluence_space" "example" {
  key = "EXAMPLE"
}

resource "confluence_page" "example" {
  space_id = data.confluence_space.example.id
  title    = "Example Page"
  body     = "<p>Page with an attachment.</p>"
}

resource "confluence_attachment" "inline" {
  page_id    = confluence_page.example.id
  title      = "example.txt"
  media_type = "text/plain"
  data       = "This is the contents of the example attachment."
}

resource "confluence_attachment" "diagram" {
  page_id    = confluence_page.example.id
  title      = "network-diagram.png"
  media_type = "image/png"
  file_path  = "${path.module}/images/network-diagram.png"
}
```

Exactly one of `data` and `file_path` must be configured. Files referenced by
`file_path` must be available during both `plan` and `apply`, including on a
remote runner. The provider calculates their SHA-256 digest automatically so
that changing the file causes a new upload without storing its contents in
Terraform state.

## Argument Reference

The following arguments are supported:

* `data` - (Optional) Inline attachment contents. Conflicts with `file_path`.

* `file_path` - (Optional) Path to a local attachment file. Conflicts with
  `data`.

* `media_type` - (Optional) The [MIME type] of the attachment. Defaults to
  `text/plain`.

* `page_id` - (Required, Forces new resource) ID of the page containing the
  attachment.

* `title` - (Required) The title (or filename) of the attachment.

## Attributes Reference

This resource exports the following attributes:

* `file_sha256` - SHA-256 digest of the attachment content.

* `media_type` - The MIME type of the attached file.

* `page_id` - ID of the page containing the attachment.

* `title` - The title (or filename) of the attachment.

* `version` - The version number of the attachment.

## Import

Attachment can be imported using the attachment id.

```
$ terraform import confluence_attachment.default {{id}}
```

[MIME type]: https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types
