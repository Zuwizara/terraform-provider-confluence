# confluence_space

Looks up a Confluence Cloud space by its key using REST API v2.

```hcl
data "confluence_space" "docs" {
  key = "DOC"
}
```

## Arguments

- `key` - (Required) Space key.

## Attributes

- `id` - Numeric Confluence space ID.
- `name` - Space name.
- `type` - Space type.
- `status` - Space status.
- `homepage_id` - ID of the space homepage.
