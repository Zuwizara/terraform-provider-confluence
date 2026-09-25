# Examples for the Confluence Terraform Provider

To successfully run any of these examples, you must provide information to
access the confluence API. To make that part easier, a template is provided.
Copy `secrets.template.env` in the root directory of this repository to
`secrets.env` and edit the values. Before running any of the examples, source
this file `source secrets.env`.

Sourcing the file sets environment variables for CONFLUENCE_CLOUD_ID,
CONFLUENCE_API_TOKEN and CONFLUENCE_SPACE. Environment variables
are one way to configure provider and resource values. If no other value is
specified, these environment variables are used as default values. The first
first two are used to configure the Confluence provider, and CONFLUENCE_SPACE is
used any time a resource needs to specify a space.

Configuring this will also provide the values you need to run acceptance tests
with `make testacc`.
