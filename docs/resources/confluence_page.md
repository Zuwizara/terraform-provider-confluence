# confluence_page

Manages a Confluence Cloud page using REST API v2.

```hcl
resource "confluence_page" "example" {
  space_id = data.confluence_space.docs.id
  title    = "Managed page"
  body     = "<p>Managed by Terraform</p>"
}
```

## Arguments

- `space_id` - (Required, Forces new resource) Numeric Confluence space ID.
- `title` - (Required) Page title.
- `body` - (Required) Page body in Confluence storage format.
- `parent_id` - (Optional) Parent page ID.

## Attributes

- `id` - Page ID.
- `version` - Current page version.
- `url` - Confluence browser URL returned by the API.

Deleting the resource moves the page to the Confluence trash. A trashed page
is treated as absent from Terraform state. Permanent deletion is intentionally
not supported.
