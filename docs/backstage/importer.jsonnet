{
  // This is the Backstage importer, and should track all catalog types under
  // this sync_id.
  sync_id: 'backstage',

  // Load pipelines.
  pipelines: [
    import 'pipelines/api-type.jsonnet',
    import 'pipelines/component-type.jsonnet',
    import 'pipelines/entries.jsonnet',
    import 'pipelines/lifecycle.jsonnet',
  ],
}
