{
  // Mark all entries as having come from this repo.
  sync_id: 'incident-io/catalog',

  // Load each of the pipelines.
  pipelines: [
    import 'feature.jsonnet',
    import 'integration.jsonnet',
    import 'team.jsonnet',
  ],
}
