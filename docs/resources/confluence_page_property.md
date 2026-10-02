# confluence_page_property

Manages one JSON content property on a Confluence Cloud page using REST API v2.
This resource manages metadata, not the Page Properties / Content Properties
table macro. It can reference either a managed page or an existing page ID.
Other property keys and the page body are not modified.

## Full-width layout

```hcl
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
```

The appearance keys target the editor and the published page respectively.
Confluence may already have created them, so this example explicitly permits
adoption. These keys are Confluence layout conventions, not a dedicated layout
API contract; verify the result in your site's editor and published view.
See [Atlassian's layout issue](https://jira.atlassian.com/browse/CONFCLOUD-72142).

## Structured values

```hcl
resource "confluence_page_property" "metadata" {
  page_id = confluence_page.example.id
  key     = "infrastructure"
  value_json = jsonencode({
    owner   = "platform"
    enabled = true
    regions = ["eu-central-1"]
  })
}
```

Set exactly one of `value` and `value_json`. `value = "true"` stores a JSON
string; `value_json = "true"` stores a boolean. An empty `value = ""` is valid.
`value_json` accepts objects, arrays, strings, numbers, booleans, and JSON null.
The provider sends JSON without converting numbers to floating point. JSON
whitespace and object key order do not cause changes; array order does.

## Arguments

- `page_id` - (Required, Forces new resource) Positive numeric Confluence page ID.
- `key` - (Required, Forces new resource) Nonblank property key, unique on the page.
- `value` - (Optional) Plain string, automatically JSON-encoded by the provider.
  Mutually exclusive with `value_json`; exactly one must be configured.
- `value_json` - (Optional) Valid JSON, usually produced with `jsonencode(...)`.
  Mutually exclusive with `value`.
- `adopt_existing` - (Optional, default `false`) During creation, allow taking
  ownership of an existing property with the same key and setting its value.
  Otherwise, an existing key produces an error with an import ID. Changing
  this option on an already managed resource does not modify the remote value.

## Attributes

- `id` - Terraform identifier in the form `<page_id>/<property_id>`.
- `property_id` - Numeric property ID assigned by Confluence, stored as a string.
- `version` - Current property version, independent of the page version.

## Lifecycle

Terraform manages the entire value for the configured key, including all fields
of a JSON object. External changes are detected on refresh and corrected on
apply. An externally deleted property is recreated on the next apply.

Destroy deletes the property, including an adopted property. It does not restore
its former value or promise a particular default layout. The page is retained.
A missing property during deletion is treated as already deleted. Each page/key
pair should be owned by only one Terraform resource/state.

Updates read the current property version before writing. Concurrent version
conflicts are returned as errors; run a fresh plan and apply after resolving
competing writers. Authentication, permission, and server failures are surfaced
as errors rather than treated as a missing property.

## Import

```sh
terraform import confluence_page_property.metadata 123456/789012
```

Both numeric IDs are required. The properties listing endpoint can be filtered
by `key` to discover the property ID. Imported JSON strings populate `value`;
other JSON types populate `value_json`. If you prefer `value_json` for a string,
switch the configuration after import; this may produce a one-time update.
`adopt_existing` is a local setting and cannot be recovered from Confluence.

## Permissions

The token needs `read:page:confluence` and `write:page:confluence`, and its account
must be able to view and edit the page. See the
[Confluence Cloud content properties API](https://developer.atlassian.com/cloud/confluence/rest/v2/api-group-content-properties/).
