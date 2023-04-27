{
  // This is used to identify which resources this config file manages. If you
  // run a sync from a CI pipeline, we advise using the name of the repository
  // it runs from, so any syncs in other repos won't interfere.
  sync_id: 'example-org/example-repo',

  pipelines: [
    // Backstage API Type
    {
      sources: [
        {
          inline: {
            entries: [
              {
                name: 'OpenAPI',
                external_id: 'openapi',
                description:
                  'An API definition in YAML or JSON format based on the OpenAPI version 2 or version 3 spec.',
              },
              {
                name: 'AsyncAPI',
                external_id: 'asyncapi',
                description:
                  'An API definition based on the AsyncAPI spec.',
              },
              {
                name: 'GraphQL',
                external_id: 'graphql',
                description:
                  'An API definition based on GraphQL schemas for consuming GraphQL based APIs.',
              },
              {
                name: 'gRPC',
                external_id: 'grpc',
                description:
                  'An API definition based on Protocol Buffers to use with gRPC.',
              },
            ],
          },
        },
      ],
      outputs: [
        {
          name: 'Backstage API Type',
          description: 'Type or format of the API.',
          type_name: 'Custom["BackstageAPIType"]',
          source: {
            name: 'name',
            external_id: 'external_id',
          },
          attributes: [
            {
              id: 'description',
              name: 'Description',
              type: 'Text',
              source: 'description',
            },
          ],
        },
      ],
    },

    // Backstage Lifecycle
    {
      sources: [
        {
          inline: {
            entries: [
              {
                name: 'Experimental',
                external_id: 'experimental',
                description:
                  'An experiment or early, non-production component, signaling that users may not prefer to consume it over other more established components, or that there are low or no reliability guarantees.',
              },
              {
                name: 'Production',
                external_id: 'production',
                description:
                  'An established, owned, maintained component.',
              },
              {
                name: 'Deprecated',
                external_id: 'deprecated',
                description:
                  'A component that is at the end of its lifecycle, and may disappear at a later point in time.',
              },
            ],
          },
        },
      ],
      outputs: [
        {
          name: 'Backstage Lifecycle',
          description: 'Component lifecycle stage.',
          type_name: 'Custom["BackstageLifecycle"]',
          source: {
            name: 'name',
            external_id: 'external_id',
          },
          attributes: [
            {
              id: 'description',
              name: 'Description',
              type: 'Text',
              source: 'description',
            },
          ],
        },
      ],
    },

    // Loaded from catalog
    {
      sources: [
        // For the purposes of the example, source data from catalog-info.yaml,
        // a multidoc YAML that contains all the Backstage entries.
        {
          'local': {
            files: [
              'catalog-info.yaml',
            ],
          },
        },
      ],
      outputs: [
        // Backstage API
        {
          name: 'Backstage API',
          description: 'APIs synced from Backstage.',
          type_name: 'Custom["BackstageAPI"]',
          source: {
            filter: 'apiVersion == "backstage.io/v1alpha1" && kind == "API"',
            name: 'metadata.name',
            external_id: 'metadata.name',
          },
          attributes: [
            {
              id: 'description',
              name: 'Description',
              type: 'Text',
              source: 'metadata.description',
            },
            {
              id: 'api-type',
              name: 'API type',
              type: 'Custom["BackstageAPIType"]',
              source: 'spec.type',
            },
            {
              id: 'owner',
              name: 'Owner',
              type: 'Custom["BackstageGroup"]',
              source: 'spec.owner',
            },
            {
              id: 'lifecycle',
              name: 'Lifecycle',
              type: 'Custom["BackstageLifecycle"]',
              source: 'spec.lifecycle',
            },
          ],
        },

        // Backstage Component
        {
          name: 'Backstage Component',
          description: 'Components synced from Backstage.',
          type_name: 'Custom["BackstageComponent"]',
          source: {
            filter: 'apiVersion == "backstage.io/v1alpha1" && kind == "Component"',
            name: 'metadata.name',
            external_id: 'metadata.name',
          },
          attributes: [
            {
              id: 'description',
              name: 'Description',
              type: 'Text',
              source: 'metadata.description',
            },
            {
              id: 'lifecycle',
              name: 'Lifecycle',
              type: 'Custom["BackstageLifecycle"]',
              source: 'spec.lifecycle',
            },
            {
              id: 'tags',
              name: 'Tags',
              type: 'String',
              array: true,
              source: 'metadata.tags',
            },
          ],
        },

        // Backstage Group
        {
          name: 'Backstage Group',
          description: 'Groups syned from Backstage.',
          type_name: 'Custom["BackstageGroup"]',
          source: {
            filter: 'apiVersion == "backstage.io/v1alpha1" && kind == "Group"',
            name: 'metadata.name',
            external_id: 'metadata.name',
          },
          attributes: [
            {
              id: 'type',
              name: 'Type',
              type: 'String',
              source: 'spec.type',
            },
            {
              id: 'parent',
              name: 'Parent',
              type: 'Custom["BackstageGroup"]',
              source: 'spec.parent',
            },
          ],
        },

        // Backstage User
        {
          name: 'Backstage User',
          description: 'Users syned from Backstage.',
          type_name: 'Custom["BackstageUser"]',
          source: {
            filter: 'apiVersion == "backstage.io/v1alpha1" && kind == "User"',
            name: 'metadata.name',
            external_id: 'metadata.name',
          },
          attributes: [
            {
              id: 'email',
              name: 'Email',
              type: 'String',
              source: 'spec.profile.email',
            },
            {
              id: 'groups',
              name: 'Groups',
              type: 'Custom["BackstageGroup"]',
              source: 'spec.memberOf',
              array: true,
            },
          ],
        },
      ],
    },
  ],
}
