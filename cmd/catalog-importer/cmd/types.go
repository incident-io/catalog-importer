package cmd

import (
	"context"
	"fmt"
	"sort"

	"github.com/alecthomas/kingpin/v2"
	"github.com/fatih/color"
	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/client"
	"github.com/pkg/errors"
	"github.com/rodaine/table"
)

type TypesOptions struct {
	APIEndpoint string
	APIKey      string
}

func (opt *TypesOptions) Bind(cmd *kingpin.CmdClause) *TypesOptions {
	cmd.Flag("api-endpoint", "Endpoint of the incident.io API").
		Default("https://api.incident.io").
		Envar("INCIDENT_ENDPOINT").
		StringVar(&opt.APIEndpoint)
	cmd.Flag("api-key", "API key for incident.io").
		Envar("INCIDENT_API_KEY").
		StringVar(&opt.APIKey)

	return opt
}

func (opt *TypesOptions) Run(ctx context.Context, logger kitlog.Logger) error {
	if opt.APIKey == "" {
		return fmt.Errorf("no API key provided as --api-key or in INCIDENT_API_KEY")
	}

	// Build incident.io client
	cl, err := client.New(ctx, opt.APIKey, opt.APIEndpoint, Version())
	if err != nil {
		return err
	}

	resp, err := cl.CatalogV2ListResourcesWithResponse(ctx)
	if err != nil {
		return errors.Wrap(err, "finding catalog resources")
	}

	headerFmt := color.New(color.Bold).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("Name", "Type name", "Category", "Description", "Value")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	ranks := map[client.CatalogResourceV2Category]int{
		client.CatalogResourceV2CategoryPrimitive: 1,
		client.CatalogResourceV2CategoryCustom:    2,
		client.CatalogResourceV2CategoryExternal:  3,
	}

	resources := resp.JSON200.Resources
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Category == resources[j].Category {
			return resources[i].Type < resources[j].Type
		}

		return ranks[resources[i].Category] < ranks[resources[j].Category]
	})

	for _, resource := range resources {
		tbl.AddRow(
			resource.Label, resource.Type, resource.Category, resource.Description, resource.ValueDocstring,
		)
	}

	tbl.Print()

	return nil
}
