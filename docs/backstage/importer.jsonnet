{
  // This is the Backstage importer, and should track all catalog types under
  // this sync_id.
  sync_id: 'backstage',

  // Load pipelines.
  pipelines: [
    import 'api-type.jsonnet',
    import 'component-type.jsonnet',
    import 'lifecycle.jsonnet',
    import 'entries.jsonnet',
  ],
}