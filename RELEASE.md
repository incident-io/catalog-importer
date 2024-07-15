# Releasing

We have a workflow for building & releasing, which can be found in [.github/workflows](.github/workflows).
This uses a tool called [goreleaser](https://goreleaser.com/), which handles most of the release process for us (creating git tag, publishing docker images, etc).

To cut a new release, simply make a new PR with the version updated in [VERSION](cmd/catalog-importer/cmd/VERSION).
