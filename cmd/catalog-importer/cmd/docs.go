package cmd

import (
	"context"

	"github.com/alecthomas/kingpin/v2"
	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/config"
)

type DocsOptions struct {
}

func (opt *DocsOptions) Bind(cmd *kingpin.CmdClause) *DocsOptions {
	return opt
}

func (opt *DocsOptions) Run(ctx context.Context, logger kitlog.Logger) error {
	OUT(`
If you want help using the catalog-importer, be sure to contact
us via your Slack Connect channel or account manager.

But first, some quicklinks that can help you get started:

  - View your catalog => https://app.incident.io/catalog
  - Importer => https://github.com/incident-io/catalog-importer
    - Docs => https://github.com/incident-io/catalog-importer/tree/master/docs
      - Examples
        - Simple: custom catalog with Teams, Features and Integrations, all from config =>
            https://github.com/incident-io/catalog-importer/blob/master/docs/simple
        - Backstage: for those already using Backstage, imports catalog-info.yaml =>
            https://github.com/incident-io/catalog-importer/blob/master/docs/backstage
      - Deployment
        - CircleCI => https://github.com/incident-io/catalog-importer/tree/master/docs#circleci
        - GitHub Actions => https://github.com/incident-io/catalog-importer/tree/master/docs#github-actions

Below is the reference config.jsonnet, documenting all config options:

`)
	config.PrettyPrint(string(config.ReferenceConfig))

	return nil
}
