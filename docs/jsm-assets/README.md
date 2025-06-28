# JSM Assets Integration

This template demonstrates how to import objects from Jira Service Management (JSM) Assets into your incident.io catalog. This example queries JSM Assets using their AQL (Assets Query Language) API and imports objects based on their object type.

## What it does

- Connects to the JSM Assets API using your Atlassian credentials
- Queries for objects matching a specific object type (defaults to "Component")
- Creates catalog entries with the object name and adds the JSM object key as an attribute
- Uses environment variables for secure credential handling

## Setup

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

## Usage

```console
catalog-importer validate --config importer.jsonnet
catalog-importer sync --config importer.jsonnet
```

This will import all objects of the "Component" type from JSM Assets into your incident.io catalog as custom catalog entries, with the JSM object key stored as an attribute for reference.

## Customization

- **Object Type**: To change from "Component" to other types (e.g., "Server", "Application", "Database"), edit the `objectType` and `aqlQuery` variables at the top of the `importer.jsonnet` file
- **Query Logic**: Modify the `aqlQuery` variable to use different AQL expressions (e.g., `'objectType = Server AND status = "Active"'`)
- **Catalog Type Names**: Customize the `name`, `description`, and `type_name` fields in the outputs section to match your organization's naming conventions
- **Additional Attributes**: You can add more attributes by extending the `attributes` array in the template to map additional JSM Assets fields

## Files

- `importer.jsonnet` - The main configuration file that defines the sync pipeline