# Configuration Guide

This guide explains how to configure the catalog importer to sync your data into incident.io. Understanding configuration is key to getting the most out of the importer.

## Configuration basics

The importer uses a **configuration file** (usually `importer.jsonnet`) that tells it:
- Where to find your data (**sources**)
- What catalog types to create (**outputs**)  
- How to transform your data to fit incident.io's catalog

```console
catalog-importer sync --config=importer.jsonnet
```

## The mental model

Think of configuration as defining **pipelines**. Each pipeline is a data flow:

```
Source Data → Pipeline → incident.io Catalog Type
    ↓            ↓              ↓
GitHub files → Transform → Service catalog
Team list   → Filter    → Team entries
API data    → Map       → Custom type
```

Each pipeline has:
- **Sources**: Where to get data (GitHub, files, APIs)
- **Outputs**: What catalog types to create and how to populate them

## Configuration structure

Here's the basic structure every configuration file needs:

```jsonnet
{
  // Unique identifier for this importer (usually repo name)
  sync_id: 'my-org/my-repo',
  
  // List of pipelines - each creates one or more catalog types
  pipelines: [
    {
      // Where to get data from
      sources: [ /* source configs */ ],
      
      // What catalog types to create  
      outputs: [ /* output configs */ ]
    }
  ]
}
```

### Sync ID
The **sync ID** is crucial - it tells incident.io which catalog entries belong to your importer. Use your repository name (like `my-org/catalog-repo`) so it's unique across your organization.

**Why it matters:** incident.io uses this to safely update and delete entries. Entries created by one sync ID won't be modified by another importer.

## A complete example

Let's look at a simple but complete configuration:

```jsonnet
{
  sync_id: 'my-org/services',
  pipelines: [
    {
      sources: [
        {
          // Load data from GitHub repositories
          github: {
            token: '$(GITHUB_TOKEN)',
            repos: ['my-org/*'],
            files: ['service-info.yaml']
          }
        }
      ],
      outputs: [
        {
          // Create a "Service" catalog type
          name: 'Service',
          type_name: 'Custom["Service"]',
          source: {
            name: '$.metadata.name',
            external_id: '$.metadata.name'
          },
          attributes: [
            {
              id: 'description', 
              name: 'Description',
              type: 'Text',
              source: '$.spec.description'
            }
          ]
        }
      ]
    }
  ]
}
```

This configuration:
1. Loads `service-info.yaml` files from all repositories in your GitHub org
2. Creates a "Service" catalog type in incident.io
3. Maps the `metadata.name` field to the service name
4. Creates a description attribute from `spec.description`

## Sources - where your data comes from

Sources tell the importer where to find your catalog data:

- **[`inline`](sources.md#inline)** - Data defined directly in the config (good for testing)
- **[`local`](sources.md#local)** - Files on your local filesystem  
- **[`github`](sources.md#github)** - Files across GitHub repositories
- **[`backstage`](sources.md#backstage)** - Pull from Backstage API
- **[`exec`](sources.md#exec)** - Run commands to generate data
- **[`graphql`](sources.md#graphql)** - Query GraphQL APIs with pagination

**→ [Full sources documentation](sources.md)**

## Outputs - your incident.io catalog types

Outputs define what catalog types to create in incident.io and how to populate them from your source data.

Key concepts:
- Each output creates one catalog type (like "Service" or "Team")
- The importer "owns" entries it creates - it can update and delete them
- Use **expressions** to map source data to catalog attributes
- Support for relationships between catalog types

**→ [Full outputs documentation](outputs.md)**

## File formats

You can write your configuration in multiple formats:

**Jsonnet (recommended):**
```jsonnet
// importer.jsonnet - supports comments and imports
local teams = import 'teams.jsonnet';
{ sync_id: 'my-org/repo', pipelines: [...] }
```

**JSON:**
```json
{
  "sync_id": "my-org/repo",
  "pipelines": [...]
}
```

**YAML:**
```yaml
sync_id: my-org/repo
pipelines: [...]
```

**Why Jsonnet?** It supports comments, imports, and functions, making complex configurations much easier to manage. Install the [VSCode extension](https://marketplace.visualstudio.com/items?itemName=Grafana.vscode-jsonnet) for the best experience.

## Next steps

- **[Sources guide](sources.md)** - Detailed guide to all data sources
- **[Outputs guide](outputs.md)** - How to create catalog types and attributes
- **[Expressions guide](expressions.md)** - Transform and filter your data
- **[Complete reference](../config/reference.jsonnet)** - All configuration options

## Common patterns

### Multiple pipelines
You can have multiple pipelines in one configuration to create different catalog types:

```jsonnet
{
  sync_id: 'my-org/catalog',
  pipelines: [
    {
      // Services pipeline
      sources: [/* service sources */],
      outputs: [/* service output */]
    },
    {
      // Teams pipeline  
      sources: [/* team sources */],
      outputs: [/* team output */]
    }
  ]
}
```

### Filtering data
Use expressions to filter which source entries go to which outputs:

```jsonnet
source: {
  filter: '$.kind == "Service"',  // Only process Service entries
  name: '$.metadata.name'
}
```

### Environment-specific config
Use environment variables for sensitive data:

```jsonnet
{
  github: {
    token: '$(GITHUB_TOKEN)',  // Reads from environment
    repos: ['my-org/*']
  }
}
```
