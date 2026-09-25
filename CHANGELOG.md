## Unreleased

BREAKING CHANGES:

- This fork is a fresh start and does not migrate state from the original
  provider or earlier versions of this fork.
- Replaced `confluence_content` with the Cloud v2-oriented
  `confluence_page` resource.
- Consolidated inline and file uploads into `confluence_attachment`. Exactly
  one of `data` or `file_path` is required; file hashes are calculated by the
  provider.
- Updated provider authentication and configuration for Confluence Cloud.

FEATURES:

- Added a Confluence Cloud space data source.
- Added page and attachment support based on the Confluence Cloud REST API v2.
