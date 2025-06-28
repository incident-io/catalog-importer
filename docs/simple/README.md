# Files

This template creates a catalog with basic types like Feature, Integration and
Team, intended to sync catalog data from JSON/YAML/Jsonnet files.

It uses an example from the incident.io team where we sync three catalog types:

- Feature, for all product features.
- Integration, all third-party product integrations.
- Team, list all Product Development teams.

The `catalog.jsonnet` file contains all the catalog data, and `importer.jsonnet`
defines pipelines and is what the `sync` command should be run with.

Out the box, this will sync catalog types for:

![Backstage catalog types created by this config](dashboard.png)

See documentation on how to use the importer at:

- https://github.com/incident-io/catalog-importer/tree/master/docs

Otherwise get started below.

## Getting started

### 1. Install the catalog-importer:

```console
brew tap incident-io/homebrew-taps
brew install catalog-importer
```

### 2. Create an API key

Create an API key from https://app.incident.io/settings/api-keys with the
following scopes:

- View catalog types and entries
- Manage catalog types and edit catalog data

Then set that token as your `INCIDENT_API_KEY` environment variable.

### 3. Sync

Now you can run a sync to import your data into the incident catalog.

```console
$ export INCIDENT_API_KEY="<token-from-above>"
$ catalog-importer sync --config importer.jsonnet --allow-delete-all

✔ Loaded config (3 pipelines, 3 sources, 3 outputs)
✔ Connected to incident.io API (https://api.incident.io/)
✔ Found 29 catalog types, with 0 that match our sync ID (backstage)

...
```

This will load the data from `catalog.jsonnet` into your incident.io catalog.

### 4. Use your real data

Now you've loaded the sample data into your catalog, you can begin filling your
data into the `catalog.jsonnet` and modifiying the type attributes and schema in
`importer.jsonnet`.

Any questions, get in touch with support@incident.io.

## JSM Assets Integration Example

The `jsm-assets-importer.jsonnet` file demonstrates how to import objects from Jira Service Management (JSM) Assets into your incident.io catalog. This example queries JSM Assets using their AQL (Assets Query Language) API and imports objects based on their object type.

### What it does

- Connects to the JSM Assets API using your Atlassian credentials
- Queries for objects matching a specific object type (defaults to "Component")
- Creates catalog entries with the object name and adds the JSM object key as an attribute
- Uses environment variables for secure credential handling

### Setup

1. **Create a Jira API token**: Generate from https://id.atlassian.com/manage-profile/security/api-tokens

2. **Get your JSM Assets workspace ID**: 
   
   Refer to the [JSM Assets API documentation](https://developer.atlassian.com/cloud/assets/rest/) for details on how to discover your workspace ID. The workspace ID will be included in the response body of various Assets API calls.

3. **Set environment variables**:

```bash
export JSM_EMAIL="your-email@company.com"
export JSM_API_TOKEN="your-api-token"
export JSM_WORKSPACE_ID="your-workspace-id"
export INCIDENT_API_KEY="your-incident-api-key"
```

### Usage

```console
catalog-importer validate --config jsm-assets-importer.jsonnet
catalog-importer sync --config jsm-assets-importer.jsonnet
```

This will import all objects of the "Component" type from JSM Assets into your incident.io catalog as custom catalog entries, with the JSM object key stored as an attribute for reference.

### Customization

- **Object Type**: To change from "Component" to other types (e.g., "Server", "Application", "Database"), edit the `objectType` variable at the top of the `jsm-assets-importer.jsonnet` file
- **Additional Attributes**: You can add more attributes by extending the `attributes` array in the template to map additional JSM Assets fields
