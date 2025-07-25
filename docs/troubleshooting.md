# Troubleshooting Guide

Having issues with the catalog importer? This guide covers the most common problems and how to solve them.

## Quick diagnostics

Start with these commands to diagnose issues:

```console
# Check your configuration is valid
catalog-importer validate --config=importer.jsonnet

# See what would be synced without making changes
catalog-importer sync --config=importer.jsonnet --dry-run

# Check connectivity to incident.io
curl -H "Authorization: Bearer $INCIDENT_API_KEY" https://api.incident.io/v1/catalog_types
```

## Common issues

### Authentication problems

**Error:** `401 Unauthorized` or `Invalid API key`

**Solutions:**
- Check your API key is set correctly: `echo $INCIDENT_API_KEY`
- Verify the key has the right permissions in [app.incident.io/settings/api-keys](https://app.incident.io/settings/api-keys):
  - View catalog types and entries
  - Manage catalog types and edit catalog data
- Ensure there are no extra spaces or newlines in your API key

### Configuration validation errors

**Error:** `Failed to parse configuration` or `Invalid Jsonnet`

**Solutions:**
- Check for syntax errors in your Jsonnet/JSON/YAML
- Validate Jsonnet syntax: `jsonnet importer.jsonnet` (if you have jsonnet installed)
- Use a JSON validator for JSON configs
- Common issues:
  - Missing commas in JSON
  - Incorrect indentation in YAML
  - Unclosed brackets or quotes

### Source data problems

**Error:** `No entries found` or `Source returned empty results`

**Solutions for GitHub source:**
- Verify your GitHub token has access to the repositories
- Check the `repos` patterns match your repositories
- Ensure `files` patterns match your catalog files
- Test with a more specific repository: `"my-org/specific-repo"` instead of `"my-org/*"`

**Solutions for local source:**
- Check file paths are relative to where you run the command
- Verify files exist: `ls -la catalog-info.yaml`
- Test glob patterns: `ls -la **/catalog-info.yaml`

**Solutions for Backstage source:**
- Verify the Backstage endpoint is accessible
- Check your token is valid for the Backstage API
- Test the endpoint manually: `curl -H "Authorization: Bearer $BACKSTAGE_TOKEN" $BACKSTAGE_ENDPOINT`

### Expression evaluation errors

**Error:** `Failed to evaluate expression` or `JavaScript error`

**Solutions:**
- Test expressions with sample data first
- Common issues:
  - `$.field` when field doesn't exist (use `_.get($.data, 'field', 'default')`)
  - Incorrect JavaScript syntax
  - Trying to access array elements that don't exist
- Debug expressions by adding temporary `console.log()` statements

### Sync issues

**Error:** `Sync ID mismatch` or `Cannot sync catalog type`

**Solutions:**
- Check if another importer already manages this catalog type
- Verify your `sync_id` is unique and consistent
- If you need to take over a catalog type, contact incident.io support

**Error:** `Entry limit exceeded` 

**Solutions:**
- The importer supports up to 50,000 entries per catalog type
- Use filters to reduce the number of entries
- Split large catalogs into multiple types
- Contact support for enterprise limits

### Performance issues

**Slow sync times:**
- Use more specific source filters to reduce data processing
- Consider breaking large configurations into multiple smaller pipelines
- For GitHub sources, use specific repository patterns instead of wildcards

## Getting help

### Debug information

When asking for help, include:

1. **Your configuration** (with sensitive data removed):
```console
# Remove tokens and save to debug-config.jsonnet
catalog-importer validate --config=debug-config.jsonnet
```

2. **Error output**:
```console
catalog-importer sync --config=importer.jsonnet --dry-run 2>&1 | tee debug-log.txt
```

3. **Environment information**:
```console
catalog-importer --version
echo "OS: $(uname -a)"
```

### Where to get help

- **GitHub Issues**: [Open an issue](https://github.com/incident-io/catalog-importer/issues/new) for bugs or feature requests
- **incident.io Support**: Contact support@incident.io for account-specific issues
- **Documentation**: Check the [complete documentation](README.md) for detailed guides

### Before opening an issue

1. Check [existing issues](https://github.com/incident-io/catalog-importer/issues) for similar problems
2. Try the troubleshooting steps above
3. Test with a minimal configuration to isolate the problem
4. Include debug information as described above

## Advanced debugging

### Enable verbose logging

Set environment variables for more detailed output:

```console
export LOG_LEVEL=debug
catalog-importer sync --config=importer.jsonnet
```

### Test individual components

**Test source data loading:**
```console
# Create a test config with just your source
{
  sync_id: 'test',
  pipelines: [{
    sources: [/* your source config */],
    outputs: [{
      name: 'Test',
      type_name: 'Custom["Test"]',
      source: { name: '$.name || $.metadata.name', external_id: '$.id || $.metadata.name' },
      attributes: []
    }]
  }]
}
```

**Test expressions:**
```javascript
// Test in a JavaScript console or Node.js
const data = { metadata: { name: "test" } };
const result = data.metadata.name;  // Your expression logic
console.log(result);
```

### Common configuration gotchas

1. **Case sensitivity**: `Custom["Service"]` vs `Custom["service"]` - these are different types
2. **Expression context**: `$` refers to the entire source entry, not just the data you expect
3. **Array handling**: Some sources return arrays, others return objects
4. **External ID requirements**: Must be unique within the catalog type
5. **Sync ID consistency**: Must be the same across runs to maintain entry ownership