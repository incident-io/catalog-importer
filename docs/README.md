# Catalog Importer Documentation

Welcome to the complete guide for the incident.io catalog importer. This documentation will help you understand, configure, and deploy the catalog importer for your organization.

## New to the catalog importer?

Start here to understand the basics and get your first import working:

1. **[Understanding the catalog importer](#understanding-the-catalog-importer)** - What it does and why you need it
2. **[Key concepts](#key-concepts)** - Essential concepts you need to know
3. **[Getting started](#getting-started)** - Your first successful import
4. **[Choose your path](#choose-your-path)** - Pick the right approach for your data

## Understanding the catalog importer

The catalog importer is a bridge between your existing catalog data and incident.io. It:

- **Pulls data** from your sources (GitHub repos, Backstage, files, APIs)
- **Transforms data** using filters and expressions to match your needs  
- **Syncs data** into incident.io's catalog, creating types and entries
- **Maintains consistency** by updating, creating, and removing entries as needed

### How it works

```
Your Data Sources → Catalog Importer → incident.io Catalog
     ↓                     ↓                  ↓
GitHub repos         Filters & maps      Service types
Backstage API        source data        Team entries  
Local files                            Feature catalog
APIs & databases                       Custom types
```

The importer uses **pipelines** - each pipeline connects specific sources to specific catalog types in incident.io.

## Key concepts

Understanding these concepts will make everything else much clearer:

### Sync ID
Every import run has a **sync ID** (like `my-org/my-repo`). This tells incident.io which entries belong to your importer, enabling safe updates and deletions.

### External ID  
Each catalog entry can have an **external ID** - a stable identifier from your source system. This ensures that if you rename an entry, incident.io knows it's the same entity.

### Sources and Outputs
- **Sources** define where your data comes from (GitHub, files, APIs)
- **Outputs** define what catalog types to create and how to populate them

### Expressions
JavaScript expressions (like `$.metadata.name`) that extract and transform data from your sources into catalog entries.

## Getting started

[api-keys]: https://app.incident.io/settings/api-keys
[jsonnet]: https://jsonnet.org/
[vscode]: https://marketplace.visualstudio.com/items?itemName=Grafana.vscode-jsonnet

The fastest way to get started is using our interactive setup:

### 1. Create your first configuration

```console
catalog-importer init
```

This creates a working configuration based on a template. Choose:
- **Simple** - Perfect for learning, uses inline data
- **Backstage** - Import existing catalog-info.yaml files

### 2. Set up authentication

Visit [app.incident.io/settings/api-keys](https://app.incident.io/settings/api-keys) and create an API key with these permissions:
- View catalog types and entries  
- Manage catalog types and edit catalog data

```console
export INCIDENT_API_KEY="your-api-key-here"
```

### 3. Test and deploy

```console
# Check your configuration is valid
catalog-importer validate --config=importer.jsonnet

# Preview what will be synced (safe)
catalog-importer sync --config=importer.jsonnet --dry-run

# Sync your data (creates/updates catalog)
catalog-importer sync --config=importer.jsonnet
```

**Success!** Your data is now synced to incident.io's catalog.

## Choose your path

Now that you have the basics working, choose the path that matches your data:

### 📁 I have catalog files in my repositories
- **[GitHub source](sources.md#github)** - Pull catalog-info.yaml files from across your GitHub org
- **[Local files](sources.md#local)** - Load files from your filesystem
- **[CI/CD integration](deploying.md)** - Automate syncing from your pipelines

### 🏗️ I use Backstage already  
- **[Backstage source](sources.md#backstage)** - Import directly from your Backstage API
- **[Backstage example](backstage/)** - Complete working configuration for Backstage users

### 🔌 I have data in APIs or databases
- **[GraphQL source](sources.md#graphql)** - Query GraphQL APIs with pagination
- **[Exec source](sources.md#exec)** - Run scripts to fetch data from any system
- **[Expression guide](expressions.md)** - Transform and filter your data

### 🛠️ I want to understand how it works
- **[Configuration guide](config.md)** - Deep dive into pipelines, sources, and outputs
- **[Reference config](../config/reference.jsonnet)** - All available options documented
- **[External IDs and aliases](aliases-and-external-ids.md)** - Advanced identity management

## Complete documentation

**Core concepts:**
- [Configuration structure](config.md) - How pipelines, sources and outputs work
- [Data sources](sources.md) - GitHub, Backstage, files, APIs, and more  
- [Catalog outputs](outputs.md) - Creating types, attributes, and relationships
- [Data transformation](expressions.md) - JavaScript expressions for filtering and mapping

**Deployment and automation:**
- [CI/CD integration](deploying.md) - GitHub Actions, CircleCI, GitLab CI examples
- [Advanced features](aliases-and-external-ids.md) - External IDs, aliases, and data relationships
- [Troubleshooting](troubleshooting.md) - Common issues and how to solve them

**Examples and templates:**
- [Simple example](simple/) - Basic inline data configuration  
- [Backstage example](backstage/) - Complete Backstage integration

## Need help?

Can't find what you're looking for? [Open an issue](https://github.com/incident-io/catalog-importer/issues/new) and we'll help you out!

**Note:** The catalog importer supports up to 50,000 entries per catalog type. Contact us if you need support for larger catalogs.
