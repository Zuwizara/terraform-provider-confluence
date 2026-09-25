# Releasing

Releases are public GitHub releases built by GoReleaser and signed for the
Terraform Registry. Creating a tag is the publication trigger; do not create a
tag until the following checklist is complete.

## One-time setup

1. Make the GitHub repository public and ensure its name remains
   `terraform-provider-confluence`.
2. Create a GPG signing key dedicated to releases.
3. Add the armored private key as the `GPG_PRIVATE_KEY` GitHub Actions secret
   and its passphrase as `PASSPHRASE`.
4. Add the matching public key to the Terraform Registry account that owns the
   `zuwizara` namespace.
5. Confirm that the repository topic includes `terraform-provider`.

## Release checklist

1. Choose the version. Because this fork deliberately breaks compatibility
   with the original provider, use a version that communicates that reset
   clearly; do not reuse an existing tag.
2. Write the version without a leading `v` to `VERSION` and move the relevant
   entries in `CHANGELOG.md` from `Unreleased` to that version.
3. Run `make release-check`. Install GoReleaser v2 and rerun the command if the
   check reports that GoReleaser validation was skipped.
4. Review the complete diff and verify that documentation contains no secrets
   or real Confluence credentials.
5. Commit the release changes, then create and push the matching tag:

   ```sh
   version=$(tr -d '[:space:]' < VERSION)
   git tag -s "v${version}" -m "v${version}"
   git push origin "v${version}"
   ```

6. Wait for the `release` workflow. Verify that the GitHub release contains a
   zip for every supported platform, the registry manifest, the SHA-256 sums,
   and the detached signature.
7. Publish or resync `zuwizara/confluence` in the Terraform Registry and test
   installation from a clean directory.

GitHub automatically exposes the repository source for every tag. The release
archives additionally include `LICENSE` and `README.md`, preserving the
Mozilla Public License 2.0 notice alongside the binaries.
