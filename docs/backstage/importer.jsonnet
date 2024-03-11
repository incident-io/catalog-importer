{
  // This is the Backstage importer, and should track all catalog types under
  // this sync_id.
  sync_id: 'backstage',

  // Load pipelines.
  pipelines: [
    import 'docs/backstage/pipelines/api-type.jsonnet',
    import 'docs/backstage/pipelines/component-type.jsonnet',
    import 'docs/backstage/pipelines/lifecycle.jsonnet',
    import 'docs/backstage/pipelines/entries.jsonnet',
  ],
}