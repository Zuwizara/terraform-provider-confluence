[![Tests](https://github.com/zuwizara/terraform-provider-confluence/actions/workflows/make-test.yml/badge.svg)](https://github.com/zuwizara/terraform-provider-confluence/actions/workflows/make-test.yml)

# Terraform Provider for Confluence

[GitHub repository](https://github.com/zuwizara/terraform-provider-confluence)

This is an unofficial community provider. It is not affiliated with, endorsed
by, or supported by Atlassian. Atlassian and Confluence are trademarks of
Atlassian.

> [!IMPORTANT]
> This provider supports **Confluence Cloud only**. Confluence Data Center and
> the discontinued Confluence Server product are not supported.

> [!WARNING]
> This provider was created entirely through AI-assisted "vibe coding". It has
> not undergone an independent security or production-readiness audit. Review
> the source code and every Terraform plan carefully, keep state and Confluence
> backups, and use it at your own risk—especially in production environments.

## Compatibility and migration

This repository is a fork of an existing Terraform provider for Confluence,
but this rewrite is intentionally a new provider rather than a drop-in
upgrade. It does not retain the old state schema or provide an automatic
migration path from the original provider or earlier versions of this fork.
In particular, `confluence_content` has been replaced by `confluence_page`,
and the provider configuration and authentication schema have changed.

Do not upgrade an existing deployment in place and expect its previous
Terraform state to remain compatible. Before adopting this version, back up
the state, update the configuration, and deliberately move or remove the old
state entries and import the existing Confluence objects into their new
resource addresses. Review the resulting plan before applying it so Terraform
does not recreate or delete existing content unexpectedly.

## Requirements

-	[Terraform](https://www.terraform.io/downloads.html)
-	[Go](https://golang.org/doc/install)

## Provider source

Use the lowercase Terraform Registry address in every module:

```hcl
terraform {
  required_providers {
    confluence = {
      source = "zuwizara/confluence"
    }
  }
}
```

## Build and install the provider

Clone this repository, enter the provider directory, build and install the provider:

```sh
$ git clone https://github.com/zuwizara/terraform-provider-confluence.git
$ cd terraform-provider-confluence
$ make install
```

## Contributing

Contributions are welcome! Please read the contribution guidelines [Contributing to Terraform - Confluence Provider](.github/CONTRIBUTING.md)

## License

This project is available under the [Mozilla Public License 2.0](LICENSE).
Source code for each published version is available from the corresponding
[GitHub release](https://github.com/zuwizara/terraform-provider-confluence/releases).
