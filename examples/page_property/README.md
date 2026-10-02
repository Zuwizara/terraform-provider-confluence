# Page properties

Creates a page with full-width layout and structured metadata.

Configure `CONFLUENCE_CLOUD_ID`, `CONFLUENCE_API_TOKEN`, and `TF_VAR_space_key`,
then run `terraform init`, `terraform plan`, and `terraform apply` with a provider
build that includes `confluence_page_property`.

The layout properties explicitly adopt existing keys, which Confluence may
create automatically. Destroy deletes those properties as well as this example
page; previous property values are not restored. Confirm the layout in both the
published page and the editor on your Confluence site.

See the [resource documentation](../../docs/resources/confluence_page_property.md)
for JSON values, import, permissions, and lifecycle behavior.
